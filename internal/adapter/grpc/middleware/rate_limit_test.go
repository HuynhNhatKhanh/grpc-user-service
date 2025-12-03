package middleware

import (
	"context"
	"testing"

	"net"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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

// mockHandler is a simple handler that returns nil
func mockHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "success", nil
}

func TestRateLimiter_WithinLimit(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstCapacity:     10,
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	// Create context with peer info
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// Make 5 requests (within limit of 10)
	for i := 0; i < 5; i++ {
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}
}

func TestRateLimiter_ExceedLimit(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 5,
		BurstCapacity:     5, // Allow 5 requests immediately
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// Make requests up to limit
	for i := 0; i < 5; i++ {
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}

	// Next request should be rate limited
	resp, err := interceptor(ctx, nil, info, mockHandler)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Contains(t, st.Message(), "rate limit exceeded")
}

func TestRateLimiter_Disabled(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 1,
		BurstCapacity:     10,  // Adequate burst capacity
		Enabled:           false, // Disabled
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// Make many requests - should all succeed because rate limiting is disabled
	for i := 0; i < 10; i++ {
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 2,
		BurstCapacity:     10,  // Adequate burst capacity
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// IP 1: Make 2 requests (at limit)
	addr1, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:12345")
	ctx1 := peer.NewContext(context.Background(), &peer.Peer{Addr: addr1})

	for i := 0; i < 2; i++ {
		resp, err := interceptor(ctx1, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}

	// IP 2: Should still be able to make requests
	addr2, _ := net.ResolveTCPAddr("tcp", "192.168.1.2:12345")
	ctx2 := peer.NewContext(context.Background(), &peer.Peer{Addr: addr2})

	resp, err := interceptor(ctx2, nil, info, mockHandler)
	require.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestRateLimiter_XForwardedFor(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 5,
		BurstCapacity:     10,  // Adequate burst capacity
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	// Create context with X-Forwarded-For header
	md := metadata.Pairs("x-forwarded-for", "203.0.113.1")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// Make requests
	for i := 0; i < 3; i++ {
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}
}

func TestRateLimiter_DifferentMethods(t *testing.T) {
	client, _ := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 2,
		BurstCapacity:     10,  // Adequate burst capacity
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	// Method 1: Make 2 requests (at limit)
	info1 := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	for i := 0; i < 2; i++ {
		resp, err := interceptor(ctx, nil, info1, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}

	// Method 2: Should have separate rate limit
	info2 := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/CreateUser",
	}

	resp, err := interceptor(ctx, nil, info2, mockHandler)
	require.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	client, mr := setupTestRedis(t)

	logger := zaptest.NewLogger(t)
	config := RateLimiterConfig{
		RequestsPerSecond: 2,
		BurstCapacity:     4,
		Enabled:           true,
	}

	rl := NewRateLimiter(client, config, logger)
	interceptor := rl.UnaryInterceptor()

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	info := &grpc.UnaryServerInfo{
		FullMethod: "/user.UserService/GetUser",
	}

	// Make 4 requests (at limit: 2 req/s * 2s = 4)
	for i := 0; i < 4; i++ {
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	}

	// Next request should fail
	_, err := interceptor(ctx, nil, info, mockHandler)
	require.Error(t, err)

	// Verify TTL is set on the key
	key := "ratelimit:tb:/user.UserService/GetUser:127.0.0.1:12345"
	ttl := mr.TTL(key)
	assert.Greater(t, ttl.Seconds(), 0.0)
	assert.LessOrEqual(t, ttl.Seconds(), 60.0) // TTL should be ~60 seconds
}
