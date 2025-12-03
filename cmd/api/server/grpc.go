package server

import (
	pb "grpc-user-service/api/gen/go/user"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/adapter/grpc/middleware"
	"grpc-user-service/internal/usecase/user"
	"grpc-user-service/pkg/logger"

	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
)

// SetupGRPC creates and configures the gRPC server
func SetupGRPC(userUC user.Usecase, l *zap.Logger, rateLimiter *middleware.RateLimiter) *grpc.Server {
	// Create gRPC server with request ID and rate limit interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.RequestIDInterceptor(),
			rateLimiter.UnaryInterceptor(),
		),
	)
	pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(userUC, l))

	return grpcServer
}
