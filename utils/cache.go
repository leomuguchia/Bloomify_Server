// File: utils/cache.go
package utils

import (
	"context"
	"log"
	"time"

	"bloomify/config"

	"github.com/go-redis/redis/v8"
)

var (
	// BookingCacheClient is the generic cache client.
	BookingCacheClient *redis.Client
	// AuthCacheClient is the dedicated client for authorization caching.
	AuthCacheClient *redis.Client
	// OTPCacheClient is the dedicated client for caching OTPs.
	OTPCacheClient *redis.Client
	// TestCacheClient is used for testing OTP retrieval.
	TestCacheClient *redis.Client
)

// InitBookingCache initializes the generic Redis cache client using the DB from AppConfig for general caching.
func InitBookingCache() {
	log.Printf("Attempting to connect to Redis (Booking Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisCacheDB)
	BookingCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisCacheDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := BookingCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Booking Cache): %v", err)
	}
	log.Println("Connected to Redis (Booking Cache) successfully.")
}

// GetBookingCacheClient returns the generic cache client.
func GetBookingCacheClient() *redis.Client {
	if BookingCacheClient == nil {
		InitBookingCache()
	}
	return BookingCacheClient
}

// InitAuthCache initializes the Redis client for authorization caching using the DB from AppConfig for auth cache.
func InitAuthCache() {
	log.Printf("Attempting to connect to Redis (Auth Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisAuthDB)
	AuthCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisAuthDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := AuthCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Auth Cache): %v", err)
	}
	log.Println("Connected to Redis (Auth Cache) successfully.")
}

// GetAuthCacheClient returns the Redis client for authorization caching.
func GetAuthCacheClient() *redis.Client {
	if AuthCacheClient == nil {
		InitAuthCache()
	}
	return AuthCacheClient
}

// InitOTPCache initializes the Redis client for OTP caching using the DB from AppConfig for OTP cache.
func InitOTPCache() {
	log.Printf("Attempting to connect to Redis (OTP Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisOTPDB)
	OTPCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisOTPDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := OTPCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (OTP Cache): %v", err)
	}
	log.Println("Connected to Redis (OTP Cache) successfully.")
}

// GetOTPCacheClient returns the Redis client for OTP caching.
func GetOTPCacheClient() *redis.Client {
	if OTPCacheClient == nil {
		InitOTPCache()
	}
	return OTPCacheClient
}

// InitTestCache initializes the Redis client for testing purposes using hard-coded values.
func InitTestCache() {
	const (
		testAddr = "localhost:6379"
		testDB   = 5
	)
	log.Printf("Attempting to connect to Redis (Test Cache) at %s using DB %d", testAddr, testDB)
	TestCacheClient = redis.NewClient(&redis.Options{
		Addr:     testAddr,
		Password: "", // No password
		DB:       testDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := TestCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Test Cache): %v", err)
	}
	log.Println("Connected to Redis (Test Cache) successfully.")
}

// GetTestCacheClient returns the Redis client for testing purposes.
func GetTestCacheClient() *redis.Client {
	if TestCacheClient == nil {
		InitTestCache()
	}
	return TestCacheClient
}

// InitRedis initializes all Redis clients at once.
func InitRedis() {
	InitBookingCache()
	InitAuthCache()
	InitOTPCache()
	InitTestCache()
	GetLogger().Sugar().Info("All Redis clients have been successfully initialized.")
}
