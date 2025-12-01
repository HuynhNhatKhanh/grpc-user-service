package infrastructure

import (
	"fmt"
	"grpc-user-service/internal/config"
	redisclient "grpc-user-service/pkg/redis"

	"go.uber.org/zap"
)

// NewRedisClient creates a new Redis client with configuration
func NewRedisClient(cfg *config.Config, l *zap.Logger) (*redisclient.Client, error) {
	redisConfig := redisclient.Config{
		Host:        cfg.Redis.Host,
		Port:        cfg.Redis.Port,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		MaxRetries:  cfg.Redis.MaxRetries,
		PoolSize:    cfg.Redis.PoolSize,
		MinIdleConn: cfg.Redis.MinIdleConn,
	}

	rdb, err := redisclient.NewClient(redisConfig, l)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rdb, nil
}
