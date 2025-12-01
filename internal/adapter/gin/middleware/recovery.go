package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a Gin middleware for panic recovery
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				log.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// Return 500 Internal Server Error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_error",
					"message": fmt.Sprintf("Internal server error: %v", err),
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
