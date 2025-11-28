package main

import (
	"context"
	"fmt"
	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/adapter/db/postgres"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/config"
	"grpc-user-service/internal/usecase/user"
	"grpc-user-service/pkg/logger"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
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
	// Load Configuration first (to get APP_ENV for logger)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// Fallback to basic logger if config fails
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize Logger with configuration
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	loggerCfg := logger.Config{
		Level:            cfg.Logger.Level,
		Format:           cfg.Logger.Format,
		OutputPath:       cfg.Logger.OutputPath,
		SlowQuerySeconds: cfg.Logger.SlowQuerySeconds,
		EnableSampling:   cfg.Logger.EnableSampling,
		ServiceName:      cfg.Logger.ServiceName,
		ServiceVersion:   cfg.Logger.ServiceVersion,
		Environment:      env,
	}

	l, err := logger.NewWithConfig(loggerCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer l.Sync() // flushes buffer, if any

	l.Info("starting application",
		zap.String("service", cfg.Logger.ServiceName),
		zap.String("version", cfg.Logger.ServiceVersion),
		zap.String("environment", env),
	)

	// Database connection with configured GORM logger
	gormLogger := logger.NewGormLoggerWithConfig(l, cfg.Logger.SlowQuerySeconds, cfg.Logger.Level)
	db, err := gorm.Open(pgdriver.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		l.Fatal("failed to connect to database", zap.Error(err))
	}

	l.Info("database connected successfully")

	repo := postgres.NewUserRepoPG(db, l)
	uc := user.New(repo, l)

	// Create gRPC server with request ID interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(logger.RequestIDInterceptor()),
	)
	pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(uc, l))

	lc := net.ListenConfig{}
	go func() {

		lis, err := lc.Listen(context.Background(), "tcp", ":"+cfg.App.GRPCPort)
		if err != nil {
			l.Fatal("failed to listen", zap.Error(err))
		}
		l.Info("gRPC server running", zap.String("port", cfg.App.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			l.Fatal("failed to serve", zap.Error(err))
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
		l.Fatal("failed to register gateway", zap.Error(err))
	}

	l.Info("REST gateway running", zap.String("port", cfg.App.HTTPPort))

	srv := &http.Server{
		Addr:    ":" + cfg.App.HTTPPort,
		Handler: mux,
		// Good practice: enforce timeouts for servers you create!
		ReadHeaderTimeout: 2 * time.Second,
	}
	return srv.ListenAndServe()
}
