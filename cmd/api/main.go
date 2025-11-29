package main

import (
	"context"
	"fmt"
	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/adapter/cache"
	"grpc-user-service/internal/adapter/db/postgres"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/adapter/grpc/middleware"
	"grpc-user-service/internal/config"
	"grpc-user-service/internal/usecase/user"
	"grpc-user-service/pkg/logger"
	redisclient "grpc-user-service/pkg/redis"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// main is the entry point of the application.
func main() {
	if err := run(); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}

// run initializes and starts the application server.
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
	defer func() {
		if err := l.Sync(); err != nil {
			l.Fatal("failed to sync logger", zap.Error(err))
		}
	}()

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

	// Initialize Redis client
	redisConfig := redisclient.Config{
		Host:        cfg.Redis.Host,
		Port:        cfg.Redis.Port,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		MaxRetries:  cfg.Redis.MaxRetries,
		PoolSize:    cfg.Redis.PoolSize,
		MinIdleConn: cfg.Redis.MinIdleConn,
	}

	rdb, err := redisclient.NewClient(redisConfig, l)
	if err != nil {
		l.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			l.Error("failed to close Redis connection", zap.Error(err))
		}
	}()

	// Initialize cache
	userCache := cache.NewRedisUserCache(
		rdb.Client,
		time.Duration(cfg.Redis.CacheTTL)*time.Second,
		l,
	)

	// Initialize repository and usecase with cache
	repo := postgres.NewUserRepoPG(db, l)
	uc := user.New(repo, userCache, l)

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(
		rdb.Client,
		middleware.RateLimiterConfig{
			RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
			WindowSeconds:     cfg.RateLimit.WindowSeconds,
			Enabled:           cfg.RateLimit.Enabled,
		},
		l,
	)

	// Create gRPC server with request ID and rate limit interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.RequestIDInterceptor(),
			rateLimiter.UnaryInterceptor(),
		),
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
		cfg.App.GRPCPort,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		l.Fatal("failed to register gateway", zap.Error(err))
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

	l.Info("REST gateway running", zap.String("port", cfg.App.HTTPPort))
	l.Info("Swagger UI available at", zap.String("url", "http://localhost:"+cfg.App.HTTPPort+"/swagger/"))

	srv := &http.Server{
		Addr:    ":" + cfg.App.HTTPPort,
		Handler: httpMux,
		// Good practice: enforce timeouts for servers you create!
		ReadHeaderTimeout: 2 * time.Second,
	}
	return srv.ListenAndServe()
}
