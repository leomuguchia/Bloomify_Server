package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	RedisAddr           string `mapstructure:"REDIS_ADDR"`
	RedisPassword       string `mapstructure:"REDIS_PASSWORD"`
	RedisBookingCacheDB int    `mapstructure:"REDIS_CACHE_DB"`
	RedisAIContextDB    int    `mapstructure:"REDIS_AI_DB"`
	RedisAuthDB         int    `mapstructure:"REDIS_AUTH_DB"`
	RedisOTPDB          int    `mapstructure:"REDIS_OTP_DB"`
	RedisProviderAuthDB int    `mapstructure:"REDIS_PROVAUTH_DB"`
	RedisFeedDB         int    `mapstructure:"FEED_DB"`

	// Google Maps API Key.
	GoogleAPIKey             string `mapstructure:"GOOGLE_API_KEY"`
	GoogleServiceAccountFile string `mapstructure:"GOOGLE_SERVICE_ACCOUNT_FILE"`
	OpenAIAPIKey             string `mapstructure:"OPENAI_KEY"`
	StripeKey                string `mapstructure:"STRIPE_KEY"`
	GeminiAPIKey             string `mapstructure:"GEMINI_KEY"`
	ExchangeRateAPIKey       string `mapstructure:"EXCHANGE_RATE_API_KEY"`
}

var AppConfig Config
var FirebaseServiceAccountKeyPath string = "config/bloom-firebase-service-account.json"
var CountryBiasMap map[string]map[string]float64

func LoadCountryBiasMap(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read country bias file: %w", err)
	}
	if err := json.Unmarshal(data, &CountryBiasMap); err != nil {
		return fmt.Errorf("failed to parse country bias JSON: %w", err)
	}
	log.Println("Successfully loaded country bias map")
	return nil
}

func LoadConfig() {
	viper.SetConfigName("c")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
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

	// country bias map from json
	LoadCountryBiasMap("config/countryBias.json")
}

func GetEnv() string {
	return AppConfig.Env
}

func IsProduction() bool {
	return GetEnv() == "production"
}
