package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration parameters for the application.
// It includes database, application server, and logger configurations.
type Config struct {
	DB     DatabaseConfig // Database connection settings
	App    AppConfig      // Application server settings
	Logger LoggerConfig   // Logger configuration
}

// DatabaseConfig holds configuration parameters for database connection.
// These settings are used to establish connection with PostgreSQL database.
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`     // Database server host
	Port     string `mapstructure:"DB_PORT"`     // Database server port
	User     string `mapstructure:"DB_USER"`     // Database username
	Password string `mapstructure:"DB_PASSWORD"` // Database password
	Name     string `mapstructure:"DB_NAME"`     // Database name
	SSLMode  string `mapstructure:"DB_SSLMODE"`  // SSL mode for database connection
}

// AppConfig holds configuration parameters for the application servers.
// It includes ports for both gRPC and HTTP servers.
type AppConfig struct {
	GRPCPort string `mapstructure:"GRPC_PORT"` // Port for gRPC server
	HTTPPort string `mapstructure:"HTTP_PORT"` // Port for HTTP REST gateway
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

	config.App.GRPCPort = viper.GetString("GRPC_PORT")
	config.App.HTTPPort = viper.GetString("HTTP_PORT")

	config.Logger.Level = viper.GetString("LOG_LEVEL")
	config.Logger.Format = viper.GetString("LOG_FORMAT")
	config.Logger.OutputPath = viper.GetString("LOG_OUTPUT_PATH")
	config.Logger.SlowQuerySeconds = viper.GetFloat64("LOG_SLOW_QUERY_SECONDS")
	config.Logger.EnableSampling = viper.GetBool("LOG_ENABLE_SAMPLING")
	config.Logger.ServiceName = viper.GetString("SERVICE_NAME")
	config.Logger.ServiceVersion = viper.GetString("SERVICE_VERSION")

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

	viper.SetDefault("GRPC_PORT", "50051")
	viper.SetDefault("HTTP_PORT", "8080")

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
}

// DSN returns the PostgreSQL Data Source Name string.
// It constructs the connection string using the configured database parameters.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode)
}
