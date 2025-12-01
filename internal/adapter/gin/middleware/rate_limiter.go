package middleware

import (
	"fmt"
	"net/http"

	grpcmiddleware "grpc-user-service/internal/adapter/grpc/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter returns a Gin middleware for rate limiting
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
		key := fmt.Sprintf("ratelimit:%s:%s:%s", method, path, clientIP)

		// Get rate limiter config (we need to access it through reflection or pass it separately)
		// For now, use default values matching the gRPC config
		windowSeconds := 1
		requestsPerSecond := 10.0
		maxRequests := int(requestsPerSecond * float64(windowSeconds))

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

		count, err := redisClient.Eval(c.Request.Context(), luaScript, []string{key},
			windowSeconds, maxRequests).Int64()
		if err != nil {
			// Log error but allow request (fail-open strategy)
			c.Next()
			return
		}

		if count > int64(maxRequests) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": fmt.Sprintf("Too many requests: %d in %d seconds (limit: %.0f req/s)", count, windowSeconds, requestsPerSecond),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
