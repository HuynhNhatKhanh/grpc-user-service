package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config represents the configuration for the logger.
// It controls log level, format, output destination, and various logging features.
type Config struct {
	Level            string  // Log level: debug, info, warn, error
	Format           string  // Log format: json, console
	OutputPath       string  // Output destination: stdout, stderr, or file path
	SlowQuerySeconds float64 // Threshold for slow query logging in seconds
	EnableSampling   bool    // Enable log sampling for production optimization
	ServiceName      string  // Service name for log identification
	ServiceVersion   string  // Service version for log identification
	Environment      string  // Environment: production, development, etc.
}

// New creates a new zap logger with basic environment-based configuration.
// If env is "production", it creates a JSON logger suitable for production.
// Otherwise, it creates a console logger with colored output for development.
// Deprecated: Use NewWithConfig for more granular control over logger configuration.
func New(env string) (*zap.Logger, error) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewWithConfig creates a new zap logger with full configuration support.
// It supports custom log levels, output formats, destinations, and sampling.
// Returns a configured logger with service metadata and context support.
func NewWithConfig(cfg Config) (*zap.Logger, error) {
	// Parse log level
	level := parseLogLevel(cfg.Level)

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Use color encoding for console format in development
	if cfg.Format == "console" && cfg.Environment != "production" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Create encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create writer syncer
	writeSyncer := getWriteSyncer(cfg.OutputPath)

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Apply sampling if enabled (production optimization)
	if cfg.EnableSampling {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second, // 1 second tick
			100,         // first 100 entries per second
			10,          // thereafter, 1 entry per 10
		)
	}

	// Create logger with caller and stacktrace
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// Add default fields
	logger = logger.With(
		zap.String("service", cfg.ServiceName),
		zap.String("version", cfg.ServiceVersion),
		zap.String("environment", cfg.Environment),
	)

	return logger, nil
}

// parseLogLevel converts a string log level to zapcore.Level.
// Defaults to InfoLevel if the provided level is not recognized.
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getWriteSyncer returns a zapcore.WriteSyncer based on the output path.
// Supports stdout, stderr, and file output with automatic log rotation.
func getWriteSyncer(outputPath string) zapcore.WriteSyncer {
	switch outputPath {
	case "stdout", "":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		// File output with rotation
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   outputPath,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   // days
			Compress:   true, // compress rotated files
		})
	}
}

// ContextKey is the type used for context keys to avoid collisions.
type ContextKey string

const (
	// RequestIDKey is the context key for storing request ID
	RequestIDKey ContextKey = "request_id"
	// TraceIDKey is the context key for storing trace ID
	TraceIDKey ContextKey = "trace_id"
	// UserIDKey is the context key for storing user ID
	UserIDKey ContextKey = "user_id"
)

// WithContext creates a new logger with context fields extracted from the context.
// It automatically adds request_id, trace_id, and user_id if present in the context.
func WithContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	fields := make([]zap.Field, 0, 3)

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			fields = append(fields, zap.String("request_id", id))
		}
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok && id != "" {
			fields = append(fields, zap.String("trace_id", id))
		}
	}

	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			fields = append(fields, zap.String("user_id", id))
		}
	}

	if len(fields) > 0 {
		return logger.With(fields...)
	}

	return logger
}

// GetRequestID extracts the request ID from the context.
// Returns an empty string if the request ID is not found or is invalid.
func GetRequestID(ctx context.Context) string {
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// GetTraceID extracts the trace ID from the context.
// Returns an empty string if the trace ID is not found or is invalid.
func GetTraceID(ctx context.Context) string {
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserID extracts the user ID from the context.
// Returns an empty string if the user ID is not found or is invalid.
func GetUserID(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}
