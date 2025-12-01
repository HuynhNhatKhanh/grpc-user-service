package infrastructure

import (
	"fmt"
	"grpc-user-service/internal/config"
	"grpc-user-service/pkg/logger"
	"time"

	"go.uber.org/zap"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewDatabase creates a new database connection with GORM configuration
func NewDatabase(cfg *config.Config, l *zap.Logger) (*gorm.DB, error) {
	// Configure GORM logger
	gormLogger := logger.NewGormLoggerWithConfig(l, cfg.Logger.SlowQuerySeconds, cfg.Logger.Level)

	// Open database connection
	db, err := gorm.Open(pgdriver.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DB.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.DB.ConnMaxIdleTime) * time.Second)

	l.Info("database connected successfully",
		zap.Int("max_open_conns", cfg.DB.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.DB.MaxIdleConns),
		zap.Int("conn_max_lifetime_seconds", cfg.DB.ConnMaxLifetime),
		zap.Int("conn_max_idle_time_seconds", cfg.DB.ConnMaxIdleTime),
	)

	return db, nil
}

// CloseDatabase closes the database connection
func CloseDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}
