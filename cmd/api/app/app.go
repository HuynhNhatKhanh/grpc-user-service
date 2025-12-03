package app

import (
	"context"
	"fmt"
	"grpc-user-service/cmd/api/di"
	"grpc-user-service/cmd/api/server"
	"grpc-user-service/internal/config"
	"grpc-user-service/pkg/logger"
	"os"
	"time"

	"go.uber.org/zap"
)

// App represents the application
type App struct {
	Config    *config.Config
	Logger    *zap.Logger
	Server    *server.Server
	Container *di.Container
}

// New creates a new application instance
func New() (*App, error) {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	l, err := initLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create DI container
	container, err := di.NewContainer(cfg, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Create server instance
	srv := server.New(cfg, l, container.UserUC, container.RateLimiter, container.GinHandler, container.RedisClient)

	return &App{
		Config:    cfg,
		Logger:    l,
		Server:    srv,
		Container: container,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			a.Logger.Error("panic recovered in application",
				zap.Any("panic", r),
				zap.Stack("stack"),
			)
		}
	}()

	env := getEnvironment()

	a.Logger.Info("starting application",
		zap.String("service", a.Config.Logger.ServiceName),
		zap.String("version", a.Config.Logger.ServiceVersion),
		zap.String("environment", env),
	)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		// Add panic recovery for server goroutine
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("server panic: %v", r)
			}
		}()

		if err := a.Server.Start(); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		a.Logger.Info("shutting down application...")
		return a.shutdown()
	case err := <-errChan:
		return err
	}
}

// shutdown gracefully shuts down the application
func (a *App) shutdown() error {
	// Create shutdown context with configurable timeout
	timeout := time.Duration(a.Config.App.ShutdownTimeoutSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	a.Logger.Info("starting graceful shutdown",
		zap.Int("timeout_seconds", a.Config.App.ShutdownTimeoutSeconds),
	)

	var errs []error

	// Shutdown HTTP server
	if a.Server.HTTP != nil {
		a.Logger.Info("shutting down HTTP server...")
		if err := a.Server.HTTP.Shutdown(shutdownCtx); err != nil {
			a.Logger.Error("failed to shutdown HTTP server", zap.Error(err))
			errs = append(errs, fmt.Errorf("HTTP shutdown: %w", err))
		}
	}

	// Shutdown Gin server
	if a.Server.Gin != nil {
		a.Logger.Info("shutting down Gin server...")
		if err := a.Server.Gin.Shutdown(shutdownCtx); err != nil {
			a.Logger.Error("failed to shutdown Gin server", zap.Error(err))
			errs = append(errs, fmt.Errorf("gin shutdown: %w", err))
		}
	}

	// Shutdown gRPC server
	if a.Server.GRPC != nil {
		a.Logger.Info("shutting down gRPC server...")
		a.Server.GRPC.GracefulStop()
	}

	// Close container resources
	if a.Container != nil {
		a.Logger.Info("closing container resources...")
		if err := a.Container.Close(); err != nil {
			a.Logger.Error("failed to close container", zap.Error(err))
			errs = append(errs, fmt.Errorf("container close: %w", err))
		}
	}

	// Sync logger
	if err := a.Logger.Sync(); err != nil {
		// Ignore sync errors for stdout/stderr
		if err.Error() != "sync /dev/stdout: invalid argument" &&
			err.Error() != "sync /dev/stderr: invalid argument" {
			a.Logger.Error("failed to sync logger", zap.Error(err))
			errs = append(errs, fmt.Errorf("logger sync: %w", err))
		}
	}

	a.Logger.Info("application shutdown complete")

	// Return aggregated errors
	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// loadConfig loads application configuration
func loadConfig() (*config.Config, error) {
	configPath := getConfigPath()
	return config.LoadConfig(configPath)
}

// initLogger initializes the application logger
func initLogger(cfg *config.Config) (*zap.Logger, error) {
	env := getEnvironment()

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

	return logger.NewWithConfig(loggerCfg)
}

// getConfigPath returns the configuration path
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "."
}

// getEnvironment returns the application environment
func getEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	return "development"
}
