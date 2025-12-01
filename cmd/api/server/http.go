package server

import (
	"context"
	"fmt"
	pb "grpc-user-service/api/gen/go/user"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SetupHTTPGateway creates and configures the HTTP gateway server
func SetupHTTPGateway(grpcAddr string, httpAddr string, l *zap.Logger) (*http.Server, error) {
	// Create gRPC-Gateway mux
	mux := runtime.NewServeMux()
	err := pb.RegisterUserServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		grpcAddr,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register gateway: %w", err)
	}

	// Create main HTTP mux to handle both API and Swagger UI
	httpMux := http.NewServeMux()

	// Serve the swagger JSON file
	httpMux.HandleFunc("/swagger/user.swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/swagger/user.swagger.json")
	})

	// Serve Swagger UI
	httpMux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/user.swagger.json"),
	))

	// Handle all other routes with gRPC Gateway mux
	httpMux.Handle("/", mux)

	l.Info("REST gateway configured", zap.String("address", httpAddr))
	l.Info("Swagger UI available at", zap.String("url", "http://localhost"+httpAddr+"/swagger/"))

	return &http.Server{
		Addr:              httpAddr,
		Handler:           httpMux,
		ReadHeaderTimeout: 2 * time.Second,
	}, nil
}
