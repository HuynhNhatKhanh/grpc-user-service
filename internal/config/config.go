package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	DB  DatabaseConfig
	App AppConfig
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

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app") // Look for app.env
	viper.SetConfigType("env")

	viper.AutomaticEnv() // Read from environment variables

	// Replace dots with underscores in env variables (e.g. DB.HOST -> DB_HOST)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay if we have env vars
	}

	var config Config

	// Bind environment variables manually for nested structs if needed,
	// or rely on mapstructure tags if using Unmarshal
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Explicitly bind for nested structs if Unmarshal doesn't pick up env vars directly without config file
	// A common pattern is to flatten the config or use mapstructure tags matching env vars directly
	// Here we use mapstructure tags in the structs which viper uses.

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
}

// DSN returns the PostgreSQL Data Source Name
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode)
}
