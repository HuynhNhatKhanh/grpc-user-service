package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	DB     DatabaseConfig
	App    AppConfig
	Logger LoggerConfig
}

// DatabaseConfig holds configuration for the database
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

// AppConfig holds configuration for the application server
type AppConfig struct {
	GRPCPort string `mapstructure:"GRPC_PORT"`
	HTTPPort string `mapstructure:"HTTP_PORT"`
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level            string  `mapstructure:"LOG_LEVEL"`
	Format           string  `mapstructure:"LOG_FORMAT"`
	OutputPath       string  `mapstructure:"LOG_OUTPUT_PATH"`
	SlowQuerySeconds float64 `mapstructure:"LOG_SLOW_QUERY_SECONDS"`
	EnableSampling   bool    `mapstructure:"LOG_ENABLE_SAMPLING"`
	ServiceName      string  `mapstructure:"SERVICE_NAME"`
	ServiceVersion   string  `mapstructure:"SERVICE_VERSION"`
}

// LoadConfig reads configuration from file or environment variables.
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

// DSN returns the PostgreSQL Data Source Name
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode)
}
