package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all configuration values.
type Config struct {
	AppPort           string `mapstructure:"APP_PORT"`
	DatabaseURL       string `mapstructure:"DATABASE_URL"`
	Env               string `mapstructure:"ENV"`
	JWTSecret         string `mapstructure:"JWT_SECRET"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
	MaxRequestsPerMin int    `mapstructure:"MAX_REQUESTS_PER_MIN"`

	// Redis configuration.
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisCacheDB  int    `mapstructure:"REDIS_CACHE_DB"`
	RedisAuthDB   int    `mapstructure:"REDIS_AUTH_DB"`
	RedisOTPDB    int    `mapstructure:"REDIS_OTP_DB"`

	// Google Maps API Key.
	GoogleAPIKey string `mapstructure:"GOOGLE_API_KEY"`
}

var AppConfig Config

func LoadConfig() {
	// Look for a config file named "config.yaml" in the current and "config" directory.
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	// Automatically use environment variables where available.
	viper.AutomaticEnv()

	// Set default values.
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_REQUESTS_PER_MIN", 100)
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_CACHE_DB", 0)
	viper.SetDefault("REDIS_AUTH_DB", 1)
	viper.SetDefault("REDIS_OTP_DB", 2)
	viper.SetDefault("DATABASE_URL", "mongodb://localhost:27017")
	viper.SetDefault("GOOGLE_API_KEY", "")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No config file found, using environment variables only")
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}

func GetEnv() string {
	return AppConfig.Env
}

func IsProduction() bool {
	return GetEnv() == "production"
}
