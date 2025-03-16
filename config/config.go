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
	RedisOTPDB    int    `mapstructure:"REDIS_OTP_DB"` // New OTP database configuration
}

var AppConfig Config

func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	// Set defaults.
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_REQUESTS_PER_MIN", 100)
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_CACHE_DB", 0)
	viper.SetDefault("REDIS_AUTH_DB", 1)
	viper.SetDefault("REDIS_OTP_DB", 2) // Default OTP DB index
	viper.SetDefault("DATABASE_URL", "mongodb://localhost:27017")

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
