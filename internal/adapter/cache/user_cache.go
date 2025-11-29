package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	domain "grpc-user-service/internal/domain/user"
)

// UserCache defines the interface for user caching operations.
type UserCache interface {
	// Get retrieves a user from cache by ID.
	// Returns nil if user is not found in cache.
	Get(ctx context.Context, id int64) (*domain.User, error)

	// Set stores a user in cache with the configured TTL.
	Set(ctx context.Context, user *domain.User) error

	// Delete removes a user from cache by ID.
	Delete(ctx context.Context, id int64) error

	// DeleteMultiple removes multiple users from cache by IDs.
	DeleteMultiple(ctx context.Context, ids ...int64) error
}

// RedisUserCache implements UserCache using Redis as the backing store.
type RedisUserCache struct {
	client *redis.Client
	ttl    time.Duration
	log    *zap.Logger
}

// NewRedisUserCache creates a new Redis-backed user cache.
func NewRedisUserCache(client *redis.Client, ttl time.Duration, log *zap.Logger) UserCache {
	return &RedisUserCache{
		client: client,
		ttl:    ttl,
		log:    log,
	}
}

// cacheKey generates a Redis key for a user ID.
func (c *RedisUserCache) cacheKey(id int64) string {
	return fmt.Sprintf("user:%d", id)
}

// Get retrieves a user from Redis cache.
func (c *RedisUserCache) Get(ctx context.Context, id int64) (*domain.User, error) {
	key := c.cacheKey(id)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss - not an error
		c.log.Debug("cache miss", zap.Int64("user_id", id))
		return nil, nil
	}
	if err != nil {
		c.log.Error("failed to get from cache", zap.Int64("user_id", id), zap.Error(err))
		return nil, err
	}

	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		c.log.Error("failed to unmarshal cached user", zap.Int64("user_id", id), zap.Error(err))
		return nil, err
	}

	c.log.Debug("cache hit", zap.Int64("user_id", id))
	return &user, nil
}

// Set stores a user in Redis cache with TTL.
func (c *RedisUserCache) Set(ctx context.Context, user *domain.User) error {
	if user == nil {
		return fmt.Errorf("cannot cache nil user")
	}

	key := c.cacheKey(user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		c.log.Error("failed to marshal user for cache", zap.Int64("user_id", user.ID), zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		c.log.Error("failed to set cache", zap.Int64("user_id", user.ID), zap.Error(err))
		return err
	}

	c.log.Debug("cached user", zap.Int64("user_id", user.ID), zap.Duration("ttl", c.ttl))
	return nil
}

// Delete removes a user from Redis cache.
func (c *RedisUserCache) Delete(ctx context.Context, id int64) error {
	key := c.cacheKey(id)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.log.Error("failed to delete from cache", zap.Int64("user_id", id), zap.Error(err))
		return err
	}

	c.log.Debug("deleted from cache", zap.Int64("user_id", id))
	return nil
}

// DeleteMultiple removes multiple users from Redis cache.
func (c *RedisUserCache) DeleteMultiple(ctx context.Context, ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = c.cacheKey(id)
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.log.Error("failed to delete multiple from cache", zap.Int("count", len(ids)), zap.Error(err))
		return err
	}

	c.log.Debug("deleted multiple from cache", zap.Int("count", len(ids)))
	return nil
}
