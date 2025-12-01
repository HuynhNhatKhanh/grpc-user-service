package server

import (
	"context"
	"fmt"
	"grpc-user-service/internal/adapter/grpc/middleware"
	"grpc-user-service/internal/config"
	"grpc-user-service/internal/usecase/user"
	"net"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server struct holds all server dependencies
type Server struct {
	Config *config.Config
	Logger *zap.Logger
	UserUC *user.Usecase
	GRPC   *grpc.Server
	HTTP   *http.Server
}

// New creates a new server instance
func New(cfg *config.Config, l *zap.Logger, userUC *user.Usecase, rateLimiter *middleware.RateLimiter) *Server {
	return &Server{
		Config: cfg,
		Logger: l,
		UserUC: userUC,
		GRPC:   SetupGRPC(userUC, l, rateLimiter),
	}
}

// Start starts both gRPC and HTTP servers
func (s *Server) Start() error {
	// Start gRPC server
	if err := s.startGRPC(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Start HTTP gateway
	if err := s.startHTTPGateway(); err != nil {
		return fmt.Errorf("failed to start HTTP gateway: %w", err)
	}

	return nil
}

// startGRPC starts the gRPC server
func (s *Server) startGRPC() error {
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", s.grpcAddress())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.Logger.Info("gRPC server running", zap.String("address", s.grpcAddress()))
	return s.GRPC.Serve(lis)
}

// grpcAddress returns the gRPC server address
func (s *Server) grpcAddress() string {
	return ":" + s.Config.App.GRPCPort
}

// httpAddress returns the HTTP server address
func (s *Server) httpAddress() string {
	return ":" + s.Config.App.HTTPPort
}

// startHTTPGateway starts the HTTP gateway server
func (s *Server) startHTTPGateway() error {
	httpServer, err := SetupHTTPGateway(s.grpcAddress(), s.httpAddress(), s.Logger)
	if err != nil {
		return err
	}

	s.HTTP = httpServer
	s.Logger.Info("REST gateway running", zap.String("address", s.httpAddress()))

	return s.HTTP.ListenAndServe()
}
