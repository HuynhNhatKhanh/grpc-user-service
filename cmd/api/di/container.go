package di

import (
	"fmt"
	"grpc-user-service/cmd/api/infrastructure"
	"grpc-user-service/internal/adapter/cache"
	"grpc-user-service/internal/adapter/db/postgres"
	"grpc-user-service/internal/adapter/grpc/middleware"
	"grpc-user-service/internal/config"
	"grpc-user-service/internal/usecase/user"
	redisclient "grpc-user-service/pkg/redis"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Container holds all application dependencies
type Container struct {
	Config      *config.Config
	Logger      *zap.Logger
	DB          *gorm.DB
	RedisClient *redisclient.Client
	UserUC      *user.Usecase
	RateLimiter *middleware.RateLimiter
}

// NewContainer creates and initializes all application dependencies
func NewContainer(cfg *config.Config, l *zap.Logger) (*Container, error) {
	// Validate configuration before initializing any dependencies
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Initialize database
	db, err := infrastructure.NewDatabase(cfg, l)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis client
	rdb, err := infrastructure.NewRedisClient(cfg, l)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize cache layer
	userCache := cache.NewRedisUserCache(
		rdb.Client,
		time.Duration(cfg.Redis.CacheTTL)*time.Second,
		l,
	)

	// Initialize repository
	repo := postgres.NewUserRepoPG(db, l)

	// Initialize use case
	userUC := user.New(repo, userCache, l)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(
		rdb.Client,
		middleware.RateLimiterConfig{
			RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
			WindowSeconds:     cfg.RateLimit.WindowSeconds,
			Enabled:           cfg.RateLimit.Enabled,
		},
		l,
	)

	return &Container{
		Config:      cfg,
		Logger:      l,
		DB:          db,
		RedisClient: rdb,
		UserUC:      userUC,
		RateLimiter: rateLimiter,
	}, nil
}

// Close closes all resources held by the container
func (c *Container) Close() error {
	var errs []error

	// Close Redis connection
	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	// Close database connection
	if c.DB != nil {
		if err := infrastructure.CloseDatabase(c.DB); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("container close errors: %v", errs)
	}

	return nil
}
