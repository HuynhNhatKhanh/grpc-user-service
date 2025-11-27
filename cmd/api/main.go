package main

import (
	"context"
	"fmt"
	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/adapter/db/postgres"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/config"
	"grpc-user-service/internal/usecase/user"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}

func run() error {
	// Load Configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Database connection
	db, err := gorm.Open(pgdriver.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	repo := postgres.NewUserRepoPG(db)
	uc := user.New(repo)

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(uc))

	lc := net.ListenConfig{}
	go func() {
		lis, err := lc.Listen(context.Background(), "tcp", ":"+cfg.App.GRPCPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("gRPC server running on :%s", cfg.App.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	mux := runtime.NewServeMux()
	err = pb.RegisterUserServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		"localhost:"+cfg.App.GRPCPort,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	log.Printf("REST gateway running on :%s", cfg.App.HTTPPort)

	srv := &http.Server{
		Addr:    ":" + cfg.App.HTTPPort,
		Handler: mux,
		// Good practice: enforce timeouts for servers you create!
		ReadHeaderTimeout: 2 * time.Second,
	}
	return srv.ListenAndServe()
}
