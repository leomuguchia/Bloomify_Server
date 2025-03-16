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
)

// InitCache initializes the generic Redis cache client using the DB from AppConfig for general caching.
func InitCache() {
	log.Printf("Attempting to connect to Redis (Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisCacheDB)
	BookingCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisCacheDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := BookingCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Cache): %v", err)
	}
	log.Println("Connected to Redis (Cache) successfully.")
}

// GetBookingCacheClient returns the generic cache client.
func GetBookingCacheClient() *redis.Client {
	if BookingCacheClient == nil {
		InitCache()
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
