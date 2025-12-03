package middleware

import (
	"fmt"
	"net/http"

	grpcmiddleware "grpc-user-service/internal/adapter/grpc/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter returns a Gin middleware for rate limiting using Token Bucket algorithm
func RateLimiter(limiter *grpcmiddleware.RateLimiter, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || redisClient == nil {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Get request method and path for rate limit key
		method := c.Request.Method
		path := c.Request.URL.Path
		// Use Token Bucket key prefix for consistency with gRPC
		key := fmt.Sprintf("ratelimit:tb:%s:%s:%s", method, path, clientIP)

		// Get rate limiter config
		// Note: We use the same config as gRPC rate limiter
		requestsPerSecond := 10.0 // Default, should match gRPC config
		burstCapacity := 20       // Default, should match gRPC config

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
				redis.call('HMSET', key, 'last_refill', now, 'tokens', tokens)
				redis.call('EXPIRE', key, 60)
				return 0  -- Deny request
			end
		`

		// Get current timestamp in seconds
		now := float64(redisClient.Time(c.Request.Context()).Val().Unix())

		allowed, err := redisClient.Eval(c.Request.Context(), luaScript, []string{key},
			requestsPerSecond,
			burstCapacity,
			now,
			1, // Always request 1 token
		).Int64()

		if err != nil {
			// Log error but allow request (fail-open strategy)
			c.Next()
			return
		}

		if allowed == 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": fmt.Sprintf("Rate limit exceeded: %.2f requests/second (burst capacity: %d)", requestsPerSecond, burstCapacity),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
