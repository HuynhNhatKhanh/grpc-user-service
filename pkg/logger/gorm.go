package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger is a custom GORM logger implementation that uses zap for structured logging.
// It provides context-aware database query logging with slow query detection.
type GormLogger struct {
	ZapLogger     *zap.Logger         // Underlying zap logger for structured logging
	SlowThreshold time.Duration       // Threshold for slow query detection
	LogLevel      gormlogger.LogLevel // Minimum log level for GORM operations
}

// NewGormLogger creates a new GORM logger with default configuration.
// It uses zap as the underlying logger with a 200ms slow query threshold.
// Deprecated: Use NewGormLoggerWithConfig for better configuration control.
func NewGormLogger(zapLogger *zap.Logger) *GormLogger {
	return &GormLogger{
		ZapLogger:     zapLogger,
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      gormlogger.Warn,
	}
}

// NewGormLoggerWithConfig creates a new GORM logger with custom configuration.
// It allows setting custom slow query threshold and log level based on string input.
func NewGormLoggerWithConfig(zapLogger *zap.Logger, slowQuerySeconds float64, logLevel string) *GormLogger {
	// Parse log level
	var level gormlogger.LogLevel
	switch logLevel {
	case "silent":
		level = gormlogger.Silent
	case "error":
		level = gormlogger.Error
	case "warn", "warning":
		level = gormlogger.Warn
	case "info", "debug":
		level = gormlogger.Info
	default:
		level = gormlogger.Warn
	}

	slowThreshold := time.Duration(slowQuerySeconds * float64(time.Second))

	return &GormLogger{
		ZapLogger:     zapLogger,
		SlowThreshold: slowThreshold,
		LogLevel:      level,
	}
}

// LogMode sets the log level for the GORM logger and returns a new instance.
// This implements the gormlogger.Interface for dynamic log level changes.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs informational messages from GORM operations.
// It includes context information like request ID if available.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		logger := WithContext(ctx, l.ZapLogger)
		logger.Sugar().Infof(msg, data...)
	}
}

// Warn logs warning messages from GORM operations.
// It includes context information like request ID if available.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		logger := WithContext(ctx, l.ZapLogger)
		logger.Sugar().Warnf(msg, data...)
	}
}

// Error logs error messages from GORM operations.
// It includes context information like request ID if available.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		logger := WithContext(ctx, l.ZapLogger)
		logger.Sugar().Errorf(msg, data...)
	}
}

// Trace logs SQL query execution details including timing, SQL statement, and row count.
// It automatically detects slow queries and logs them as warnings.
// SQL statements are truncated if too long to prevent log flooding.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Truncate SQL if too long (prevent log flooding)
	const maxSQLLength = 1000
	truncated := false
	if len(sql) > maxSQLLength {
		sql = sql[:maxSQLLength] + "..."
		truncated = true
	}

	// Get logger with context (includes request_id if available)
	logger := WithContext(ctx, l.ZapLogger)

	// Base fields
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
		zap.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	if truncated {
		fields = append(fields, zap.Bool("sql_truncated", true))
	}

	// Log errors (except ErrRecordNotFound which is not really an error)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fields = append(fields, zap.Error(err))
		logger.Error("gorm query error", fields...)
		return
	}

	// Log slow queries as warnings
	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn {
		fields = append(fields, zap.Duration("threshold", l.SlowThreshold))
		logger.Warn("gorm slow query", fields...)
		return
	}

	// Log all queries at info level
	if l.LogLevel >= gormlogger.Info {
		logger.Info("gorm query", fields...)
	}
}
