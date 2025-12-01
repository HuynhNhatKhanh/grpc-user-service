package server

import (
	"net/http"
	"time"

	ginhandler "grpc-user-service/internal/adapter/gin/handler"
	ginrouter "grpc-user-service/internal/adapter/gin/router"
	grpcmiddleware "grpc-user-service/internal/adapter/grpc/middleware"
	redisclient "grpc-user-service/pkg/redis"

	"go.uber.org/zap"
)

// SetupGinServer creates and configures the Gin REST API server
func SetupGinServer(
	handler *ginhandler.UserHandler,
	rateLimiter *grpcmiddleware.RateLimiter,
	redisClient *redisclient.Client,
	ginAddr string,
	l *zap.Logger,
) (*http.Server, error) {
	// Setup Gin router with all middleware and routes
	router := ginrouter.SetupRouter(handler, rateLimiter, redisClient, l)

	l.Info("Gin REST API configured", zap.String("address", ginAddr))

	return &http.Server{
		Addr:              ginAddr,
		Handler:           router,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}, nil
}
