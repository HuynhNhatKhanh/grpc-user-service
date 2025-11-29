package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Config holds Redis connection configuration.
type Config struct {
	Host        string
	Port        string
	Password    string
	DB          int
	MaxRetries  int
	PoolSize    int
	MinIdleConn int
}

// Client wraps redis.Client with additional functionality.
type Client struct {
	*redis.Client
	log *zap.Logger
}

// NewClient creates a new Redis client with the provided configuration.
// It establishes a connection pool and verifies connectivity with a ping.
func NewClient(cfg Config, log *zap.Logger) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConn,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	log.Info("Redis connected successfully",
		zap.String("addr", addr),
		zap.Int("db", cfg.DB),
		zap.Int("pool_size", cfg.PoolSize),
	)

	return &Client{
		Client: rdb,
		log:    log,
	}, nil
}

// Ping checks if the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}

// Close gracefully closes the Redis connection.
func (c *Client) Close() error {
	c.log.Info("Closing Redis connection")
	return c.Client.Close()
}
