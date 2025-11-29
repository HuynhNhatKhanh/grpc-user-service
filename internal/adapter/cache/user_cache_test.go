package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	domain "grpc-user-service/internal/domain/user"
)

// setupTestRedis creates a miniredis instance for testing
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client, mr
}

func TestRedisUserCache_Set_Success(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	user := &domain.User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	err := cache.Set(context.Background(), user)
	require.NoError(t, err)

	// Verify data is in Redis
	data, err := client.Get(context.Background(), "user:1").Bytes()
	require.NoError(t, err)

	var cached domain.User
	err = json.Unmarshal(data, &cached)
	require.NoError(t, err)

	assert.Equal(t, user.ID, cached.ID)
	assert.Equal(t, user.Name, cached.Name)
	assert.Equal(t, user.Email, cached.Email)
}

func TestRedisUserCache_Set_NilUser(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	err := cache.Set(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cache nil user")
}

func TestRedisUserCache_Get_Success(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	// Set user first
	user := &domain.User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	err := cache.Set(context.Background(), user)
	require.NoError(t, err)

	// Get user
	cached, err := cache.Get(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, cached)

	assert.Equal(t, user.ID, cached.ID)
	assert.Equal(t, user.Name, cached.Name)
	assert.Equal(t, user.Email, cached.Email)
}

func TestRedisUserCache_Get_CacheMiss(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	// Get non-existent user
	cached, err := cache.Get(context.Background(), 999)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestRedisUserCache_Delete_Success(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	// Set user first
	user := &domain.User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	err := cache.Set(context.Background(), user)
	require.NoError(t, err)

	// Delete user
	err = cache.Delete(context.Background(), 1)
	require.NoError(t, err)

	// Verify user is deleted
	cached, err := cache.Get(context.Background(), 1)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestRedisUserCache_DeleteMultiple_Success(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	// Set multiple users
	users := []*domain.User{
		{ID: 1, Name: "User 1", Email: "user1@example.com"},
		{ID: 2, Name: "User 2", Email: "user2@example.com"},
		{ID: 3, Name: "User 3", Email: "user3@example.com"},
	}

	for _, user := range users {
		err := cache.Set(context.Background(), user)
		require.NoError(t, err)
	}

	// Delete multiple users
	err := cache.DeleteMultiple(context.Background(), 1, 2, 3)
	require.NoError(t, err)

	// Verify all users are deleted
	for _, user := range users {
		cached, err := cache.Get(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Nil(t, cached)
	}
}

func TestRedisUserCache_DeleteMultiple_EmptyIDs(t *testing.T) {
	client, _ := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	cache := NewRedisUserCache(client, 5*time.Minute, logger)

	// Delete with empty IDs should not error
	err := cache.DeleteMultiple(context.Background())
	require.NoError(t, err)
}

func TestRedisUserCache_TTL(t *testing.T) {
	client, mr := setupTestRedis(t)


	logger := zaptest.NewLogger(t)
	ttl := 2 * time.Second
	cache := NewRedisUserCache(client, ttl, logger)

	user := &domain.User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	err := cache.Set(context.Background(), user)
	require.NoError(t, err)

	// Fast forward time in miniredis
	mr.FastForward(3 * time.Second)

	// User should be expired
	cached, err := cache.Get(context.Background(), 1)
	require.NoError(t, err)
	assert.Nil(t, cached)
}
