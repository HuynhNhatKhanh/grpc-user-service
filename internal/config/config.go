package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration parameters for the application.
// It includes database, application server, and logger configurations.
type Config struct {
	DB        DatabaseConfig  // Database connection settings
	App       AppConfig       // Application server settings
	Logger    LoggerConfig    // Logger configuration
	Redis     RedisConfig     // Redis connection settings
	RateLimit RateLimitConfig // Rate limiting configuration
}

// DatabaseConfig holds configuration parameters for database connection.
// These settings are used to establish connection with PostgreSQL database.
type DatabaseConfig struct {
	Host            string `mapstructure:"DB_HOST"`               // Database server host
	Port            string `mapstructure:"DB_PORT"`               // Database server port
	User            string `mapstructure:"DB_USER"`               // Database username
	Password        string `mapstructure:"DB_PASSWORD"`           // Database password
	Name            string `mapstructure:"DB_NAME"`               // Database name
	SSLMode         string `mapstructure:"DB_SSLMODE"`            // SSL mode for database connection
	MaxOpenConns    int    `mapstructure:"DB_MAX_OPEN_CONNS"`     // Maximum number of open connections
	MaxIdleConns    int    `mapstructure:"DB_MAX_IDLE_CONNS"`     // Maximum number of idle connections
	ConnMaxLifetime int    `mapstructure:"DB_CONN_MAX_LIFETIME"`  // Maximum lifetime of a connection in seconds
	ConnMaxIdleTime int    `mapstructure:"DB_CONN_MAX_IDLE_TIME"` // Maximum idle time of a connection in seconds
}

// AppConfig holds configuration parameters for the application servers.
// It includes ports for both gRPC and HTTP servers.
type AppConfig struct {
	GRPCPort               string `mapstructure:"GRPC_PORT"`                // Port for gRPC server
	HTTPPort               string `mapstructure:"HTTP_PORT"`                // Port for HTTP REST gateway
	ShutdownTimeoutSeconds int    `mapstructure:"SHUTDOWN_TIMEOUT_SECONDS"` // Graceful shutdown timeout in seconds
}

// LoggerConfig holds configuration parameters for the logging system.
// It controls log level, format, output destination, and sampling behavior.
type LoggerConfig struct {
	Level            string  `mapstructure:"LOG_LEVEL"`              // Log level (debug, info, warn, error)
	Format           string  `mapstructure:"LOG_FORMAT"`             // Log format (console, json)
	OutputPath       string  `mapstructure:"LOG_OUTPUT_PATH"`        // Log output destination
	SlowQuerySeconds float64 `mapstructure:"LOG_SLOW_QUERY_SECONDS"` // Threshold for slow query logging
	EnableSampling   bool    `mapstructure:"LOG_ENABLE_SAMPLING"`    // Enable log sampling for high traffic
	ServiceName      string  `mapstructure:"SERVICE_NAME"`           // Service name for log identification
	ServiceVersion   string  `mapstructure:"SERVICE_VERSION"`        // Service version for log identification
}

// RedisConfig holds configuration parameters for Redis connection.
// These settings are used to establish connection with Redis for caching and rate limiting.
type RedisConfig struct {
	Host        string `mapstructure:"REDIS_HOST"`              // Redis server host
	Port        string `mapstructure:"REDIS_PORT"`              // Redis server port
	Password    string `mapstructure:"REDIS_PASSWORD"`          // Redis password (empty for no auth)
	DB          int    `mapstructure:"REDIS_DB"`                // Redis database number
	CacheTTL    int    `mapstructure:"REDIS_CACHE_TTL_SECONDS"` // Cache TTL in seconds
	MaxRetries  int    `mapstructure:"REDIS_MAX_RETRIES"`       // Maximum number of retries
	PoolSize    int    `mapstructure:"REDIS_POOL_SIZE"`         // Connection pool size
	MinIdleConn int    `mapstructure:"REDIS_MIN_IDLE_CONN"`     // Minimum idle connections
}

// RateLimitConfig holds configuration parameters for rate limiting.
// It controls how many requests are allowed per time window.
type RateLimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"RATE_LIMIT_REQUESTS_PER_SECOND"` // Maximum requests per second
	WindowSeconds     int     `mapstructure:"RATE_LIMIT_WINDOW_SECONDS"`      // Time window in seconds
	Enabled           bool    `mapstructure:"RATE_LIMIT_ENABLED"`             // Enable/disable rate limiting
}

// LoadConfig reads configuration from file or environment variables.
// It first sets default values, then attempts to read from app.env file,
// and finally overrides with any environment variables that are set.
// Returns a populated Config struct or an error if configuration is invalid.
func LoadConfig(path string) (*Config, error) {
	// Set defaults first
	setDefaults()

	viper.AddConfigPath(path)
	viper.SetConfigName("app") // Look for app.env
	viper.SetConfigType("env")

	viper.AutomaticEnv() // Read from environment variables

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay if we have env vars
	}

	var config Config

	// Manually populate config from viper
	config.DB.Host = viper.GetString("DB_HOST")
	config.DB.Port = viper.GetString("DB_PORT")
	config.DB.User = viper.GetString("DB_USER")
	config.DB.Password = viper.GetString("DB_PASSWORD")
	config.DB.Name = viper.GetString("DB_NAME")
	config.DB.SSLMode = viper.GetString("DB_SSLMODE")
	config.DB.MaxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
	config.DB.MaxIdleConns = viper.GetInt("DB_MAX_IDLE_CONNS")
	config.DB.ConnMaxLifetime = viper.GetInt("DB_CONN_MAX_LIFETIME")
	config.DB.ConnMaxIdleTime = viper.GetInt("DB_CONN_MAX_IDLE_TIME")

	config.App.GRPCPort = viper.GetString("GRPC_PORT")
	config.App.HTTPPort = viper.GetString("HTTP_PORT")
	config.App.ShutdownTimeoutSeconds = viper.GetInt("SHUTDOWN_TIMEOUT_SECONDS")

	config.Logger.Level = viper.GetString("LOG_LEVEL")
	config.Logger.Format = viper.GetString("LOG_FORMAT")
	config.Logger.OutputPath = viper.GetString("LOG_OUTPUT_PATH")
	config.Logger.SlowQuerySeconds = viper.GetFloat64("LOG_SLOW_QUERY_SECONDS")
	config.Logger.EnableSampling = viper.GetBool("LOG_ENABLE_SAMPLING")
	config.Logger.ServiceName = viper.GetString("SERVICE_NAME")
	config.Logger.ServiceVersion = viper.GetString("SERVICE_VERSION")

	config.Redis.Host = viper.GetString("REDIS_HOST")
	config.Redis.Port = viper.GetString("REDIS_PORT")
	config.Redis.Password = viper.GetString("REDIS_PASSWORD")
	config.Redis.DB = viper.GetInt("REDIS_DB")
	config.Redis.CacheTTL = viper.GetInt("REDIS_CACHE_TTL_SECONDS")
	config.Redis.MaxRetries = viper.GetInt("REDIS_MAX_RETRIES")
	config.Redis.PoolSize = viper.GetInt("REDIS_POOL_SIZE")
	config.Redis.MinIdleConn = viper.GetInt("REDIS_MIN_IDLE_CONN")

	config.RateLimit.RequestsPerSecond = viper.GetFloat64("RATE_LIMIT_REQUESTS_PER_SECOND")
	config.RateLimit.WindowSeconds = viper.GetInt("RATE_LIMIT_WINDOW_SECONDS")
	config.RateLimit.Enabled = viper.GetBool("RATE_LIMIT_ENABLED")

	return &config, nil
}

// setDefaults defines default configuration values for all settings.
// These values are used when no configuration file or environment variables are provided.
func setDefaults() {
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "grpc_user_service")
	viper.SetDefault("DB_SSLMODE", "disable")
	// Database connection pool defaults
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", 300)  // 5 minutes in seconds
	viper.SetDefault("DB_CONN_MAX_IDLE_TIME", 600) // 10 minutes in seconds

	viper.SetDefault("GRPC_PORT", "50051")
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("SHUTDOWN_TIMEOUT_SECONDS", 30)

	// Logger defaults
	env := viper.GetString("APP_ENV")
	if env == "production" {
		viper.SetDefault("LOG_LEVEL", "info")
		viper.SetDefault("LOG_FORMAT", "json")
		viper.SetDefault("LOG_ENABLE_SAMPLING", true)
	} else {
		viper.SetDefault("LOG_LEVEL", "debug")
		viper.SetDefault("LOG_FORMAT", "console")
		viper.SetDefault("LOG_ENABLE_SAMPLING", false)
	}
	viper.SetDefault("LOG_OUTPUT_PATH", "stdout")
	viper.SetDefault("LOG_SLOW_QUERY_SECONDS", 0.2)
	viper.SetDefault("SERVICE_NAME", "grpc-user-service")
	viper.SetDefault("SERVICE_VERSION", "1.0.0")

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_CACHE_TTL_SECONDS", 300) // 5 minutes
	viper.SetDefault("REDIS_MAX_RETRIES", 3)
	viper.SetDefault("REDIS_POOL_SIZE", 10)
	viper.SetDefault("REDIS_MIN_IDLE_CONN", 5)

	// Rate limit defaults
	viper.SetDefault("RATE_LIMIT_REQUESTS_PER_SECOND", 10.0)
	viper.SetDefault("RATE_LIMIT_WINDOW_SECONDS", 1)
	viper.SetDefault("RATE_LIMIT_ENABLED", true)
}

// Validate validates all configuration parameters.
// It checks for required fields, valid ranges, and logical consistency.
// Returns an error if any validation fails.
func (c *Config) Validate() error {
	if err := c.DB.Validate(); err != nil {
		return err
	}
	if err := c.App.Validate(); err != nil {
		return err
	}
	if err := c.Logger.Validate(); err != nil {
		return err
	}
	if err := c.Redis.Validate(); err != nil {
		return err
	}
	if err := c.RateLimit.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate validates database configuration
func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Port == "" {
		return fmt.Errorf("DB_PORT is required")
	}
	if err := validatePort(c.Port); err != nil {
		return fmt.Errorf("DB_PORT is invalid: %w", err)
	}
	if c.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be positive, got %d", c.MaxOpenConns)
	}
	if c.MaxIdleConns <= 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be positive, got %d", c.MaxIdleConns)
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)",
			c.MaxIdleConns, c.MaxOpenConns)
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must be positive, got %d", c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_IDLE_TIME must be positive, got %d", c.ConnMaxIdleTime)
	}
	return nil
}

// Validate validates app configuration
func (c *AppConfig) Validate() error {
	if c.GRPCPort == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}
	if err := validatePort(c.GRPCPort); err != nil {
		return fmt.Errorf("GRPC_PORT is invalid: %w", err)
	}
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT is required")
	}
	if err := validatePort(c.HTTPPort); err != nil {
		return fmt.Errorf("HTTP_PORT is invalid: %w", err)
	}
	if c.ShutdownTimeoutSeconds <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT_SECONDS must be positive, got %d", c.ShutdownTimeoutSeconds)
	}
	if c.ShutdownTimeoutSeconds > 300 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT_SECONDS cannot exceed 300 seconds (5 minutes), got %d", c.ShutdownTimeoutSeconds)
	}
	return nil
}

// Validate validates logger configuration
func (c *LoggerConfig) Validate() error {
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Level] {
		return fmt.Errorf("LOG_LEVEL must be one of [debug, info, warn, error], got %s", c.Level)
	}
	validLogFormats := map[string]bool{"console": true, "json": true}
	if !validLogFormats[c.Format] {
		return fmt.Errorf("LOG_FORMAT must be one of [console, json], got %s", c.Format)
	}
	if c.SlowQuerySeconds < 0 {
		return fmt.Errorf("LOG_SLOW_QUERY_SECONDS cannot be negative, got %f", c.SlowQuerySeconds)
	}
	if c.ServiceName == "" {
		return fmt.Errorf("SERVICE_NAME is required")
	}
	if c.ServiceVersion == "" {
		return fmt.Errorf("SERVICE_VERSION is required")
	}
	return nil
}

// Validate validates Redis configuration
func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if c.Port == "" {
		return fmt.Errorf("REDIS_PORT is required")
	}
	if err := validatePort(c.Port); err != nil {
		return fmt.Errorf("REDIS_PORT is invalid: %w", err)
	}
	if c.DB < 0 {
		return fmt.Errorf("REDIS_DB cannot be negative, got %d", c.DB)
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("REDIS_CACHE_TTL_SECONDS must be positive, got %d", c.CacheTTL)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("REDIS_MAX_RETRIES cannot be negative, got %d", c.MaxRetries)
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("REDIS_POOL_SIZE must be positive, got %d", c.PoolSize)
	}
	if c.MinIdleConn < 0 {
		return fmt.Errorf("REDIS_MIN_IDLE_CONN cannot be negative, got %d", c.MinIdleConn)
	}
	if c.MinIdleConn > c.PoolSize {
		return fmt.Errorf("REDIS_MIN_IDLE_CONN (%d) cannot exceed REDIS_POOL_SIZE (%d)",
			c.MinIdleConn, c.PoolSize)
	}
	return nil
}

// Validate validates rate limit configuration
func (c *RateLimitConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.RequestsPerSecond <= 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_SECOND must be positive when rate limiting is enabled, got %f",
			c.RequestsPerSecond)
	}
	if c.WindowSeconds <= 0 {
		return fmt.Errorf("RATE_LIMIT_WINDOW_SECONDS must be positive when rate limiting is enabled, got %d",
			c.WindowSeconds)
	}
	return nil
}

// validatePort checks if a port string is a valid port number (1-65535).
func validatePort(port string) error {
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return fmt.Errorf("port must be a number, got %s", port)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", portNum)
	}
	return nil
}

// DSN returns the PostgreSQL Data Source Name string.
// It constructs the connection string using the configured database parameters.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode)
}
