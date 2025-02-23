package utils

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration values
type Config struct {
	AppPort           string `mapstructure:"APP_PORT"`
	DatabaseURL       string `mapstructure:"DATABASE_URL"`
	Env               string `mapstructure:"ENV"`
	JWTSecret         string `mapstructure:"JWT_SECRET"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
	MaxRequestsPerMin int    `mapstructure:"MAX_REQUESTS_PER_MIN"`
}

// Global variable to store configuration
var AppConfig Config

// LoadConfig initializes Viper to load config values from env, file, or defaults
func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Read environment variables
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_REQUESTS_PER_MIN", 100)

	// Read configuration file if available
	if err := viper.ReadInConfig(); err != nil {
		log.Println("No config file found, using environment variables only")
	}

	// Unmarshal into AppConfig struct
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}

// GetEnv returns the application environment
func GetEnv() string {
	return AppConfig.Env
}

// IsProduction checks if the environment is production
func IsProduction() bool {
	return GetEnv() == "production"
}
