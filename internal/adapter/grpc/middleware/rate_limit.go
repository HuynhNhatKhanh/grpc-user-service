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

// RateLimiterConfig holds configuration for the Token Bucket rate limiter.
type RateLimiterConfig struct {
	RequestsPerSecond float64 // Token refill rate (tokens per second)
	BurstCapacity     int     // Maximum tokens in bucket (allows burst traffic)
	Enabled           bool
}

// RateLimiter implements gRPC rate limiting using Token Bucket algorithm with Redis.
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

		// Create rate limit key: ratelimit:tb:{method}:{ip}
		key := fmt.Sprintf("ratelimit:tb:%s:%s", info.FullMethod, clientIP)

		// Token Bucket algorithm implemented in Lua for atomicity
		// Data structure: {last_refill_time, current_tokens}
		luaScript := `
			local key = KEYS[1]
			local rate = tonumber(ARGV[1])         -- tokens per second
			local capacity = tonumber(ARGV[2])     -- max tokens in bucket
			local now = tonumber(ARGV[3])          -- current timestamp
			local requested = tonumber(ARGV[4])    -- tokens requested (always 1)
			
			-- Get current bucket state
			local bucket = redis.call('HMGET', key, 'last_refill', 'tokens')
			local last_refill = tonumber(bucket[1]) or now
			local tokens = tonumber(bucket[2]) or capacity
			
			-- Calculate tokens to add based on elapsed time
			local elapsed = math.max(0, now - last_refill)
			local tokens_to_add = elapsed * rate
			tokens = math.min(capacity, tokens + tokens_to_add)
			
			-- Try to consume requested tokens
			if tokens >= requested then
				-- Success: consume token
				tokens = tokens - requested
				redis.call('HMSET', key, 'last_refill', now, 'tokens', tokens)
				redis.call('EXPIRE', key, 60)  -- Keep bucket for 60 seconds
				return 1  -- Allow request
			else
				-- Failure: not enough tokens
				-- Still update last_refill to prevent token accumulation during rate limit
				redis.call('HMSET', key, 'last_refill', now, 'tokens', tokens)
				redis.call('EXPIRE', key, 60)
				return 0  -- Deny request
			end
		`

		// Execute Lua script
		// Get current timestamp in seconds (floating point for precision)
		now := float64(rl.client.Time(ctx).Val().Unix())

		allowed, err := rl.client.Eval(ctx, luaScript, []string{key},
			rl.config.RequestsPerSecond,
			rl.config.BurstCapacity,
			now,
			1, // Always request 1 token
		).Int64()

		if err != nil {
			// On Redis error, allow request to proceed (fail open)
			rl.log.Warn("rate limiter redis error, allowing request",
				zap.String("client_ip", clientIP),
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return handler(ctx, req)
		}

		// Check if request is allowed
		if allowed == 0 {
			rl.log.Warn("rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.String("method", info.FullMethod),
				zap.Float64("rate", rl.config.RequestsPerSecond),
				zap.Int("burst_capacity", rl.config.BurstCapacity),
			)
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: %.2f requests/second (burst capacity: %d)",
				rl.config.RequestsPerSecond, rl.config.BurstCapacity)
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
