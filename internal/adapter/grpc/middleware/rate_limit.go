package middleware

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiterConfig holds configuration for the rate limiter.
type RateLimiterConfig struct {
	RequestsPerSecond float64
	WindowSeconds     int
	Enabled           bool
}

// RateLimiter implements gRPC rate limiting using Redis.
type RateLimiter struct {
	client *redis.Client
	config RateLimiterConfig
	log    *zap.Logger
}

// NewRateLimiter creates a new rate limiter interceptor.
func NewRateLimiter(client *redis.Client, config RateLimiterConfig, log *zap.Logger) *RateLimiter {
	return &RateLimiter{
		client: client,
		config: config,
		log:    log,
	}
}

// UnaryInterceptor returns a gRPC unary interceptor for rate limiting.
func (rl *RateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip rate limiting if disabled
		if !rl.config.Enabled {
			return handler(ctx, req)
		}

		// Get client IP from peer info
		clientIP := rl.getClientIP(ctx)

		// Create rate limit key: ratelimit:{method}:{ip}
		key := fmt.Sprintf("ratelimit:%s:%s", info.FullMethod, clientIP)

		// Calculate maximum requests allowed in the window
		maxRequests := int(rl.config.RequestsPerSecond * float64(rl.config.WindowSeconds))

		// Use Redis Lua script for atomic increment with expiry
		luaScript := `
			local key = KEYS[1]
			local window = tonumber(ARGV[1])
			local max_requests = tonumber(ARGV[2])
			
			local count = redis.call('INCR', key)
			if count == 1 then
				redis.call('EXPIRE', key, window)
			end
			
			return count
		`

		count, err := rl.client.Eval(ctx, luaScript, []string{key},
			rl.config.WindowSeconds, maxRequests).Int64()
		if err != nil {
			// On Redis error, allow request to proceed (fail open)
			rl.log.Warn("rate limiter redis error, allowing request",
				zap.String("client_ip", clientIP),
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return handler(ctx, req)
		}

		// Check if limit exceeded
		if count > int64(maxRequests) {
			rl.log.Warn("rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.String("method", info.FullMethod),
				zap.Int64("count", count),
				zap.Float64("limit", rl.config.RequestsPerSecond),
			)
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: %d requests in %d seconds (limit: %.0f req/s)",
				count, rl.config.WindowSeconds, rl.config.RequestsPerSecond)
		}

		// Allow request
		return handler(ctx, req)
	}
}

// getClientIP extracts the client IP address from the gRPC context.
func (rl *RateLimiter) getClientIP(ctx context.Context) string {
	// Try to get IP from X-Forwarded-For header (for requests through gateway)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return xff[0]
		}
		if xri := md.Get("x-real-ip"); len(xri) > 0 {
			return xri[0]
		}
	}

	// Fallback to peer address
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	return "unknown"
}
