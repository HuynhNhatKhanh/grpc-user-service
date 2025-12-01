package router

import (
	"net/http"

	"grpc-user-service/internal/adapter/gin/handler"
	"grpc-user-service/internal/adapter/gin/middleware"
	grpcmiddleware "grpc-user-service/internal/adapter/grpc/middleware"
	redisclient "grpc-user-service/pkg/redis"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter configures and returns a Gin router with all routes and middleware
func SetupRouter(
	userHandler *handler.UserHandler,
	rateLimiter *grpcmiddleware.RateLimiter,
	redisClient *redisclient.Client,
	log *zap.Logger,
) *gin.Engine {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Logger(log))
	router.Use(middleware.RateLimiter(rateLimiter, redisClient.Client))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "grpc-user-service-gin",
		})
	})

	// API v1 routes
	v1 := router.Group("/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	return router
}
