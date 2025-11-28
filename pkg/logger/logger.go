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

// Config represents logger configuration
type Config struct {
	Level            string  // debug, info, warn, error
	Format           string  // json, console
	OutputPath       string  // stdout, stderr, or file path
	SlowQuerySeconds float64 // slow query threshold
	EnableSampling   bool    // enable sampling for production
	ServiceName      string  // service name for logs
	ServiceVersion   string  // service version for logs
	Environment      string  // environment (production, development, etc.)
}

// New creates a new zap logger.
// If env is "production", it creates a JSON logger.
// Otherwise, it creates a console logger.
// Deprecated: Use NewWithConfig for more control
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

// NewWithConfig creates a new zap logger with full configuration
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

// parseLogLevel converts string log level to zapcore.Level
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

// getWriteSyncer returns write syncer based on output path
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

// ContextKey is the type for context keys
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
)

// WithContext creates a logger with context fields (request_id, trace_id, user_id)
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

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}
