// File: utils/cache.go
package utils

import (
	"bloomify/config"
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// CacheClient is the generic cache client.
	CacheClient *redis.Client
	// AuthCacheClient is the dedicated client for authorization caching.
	AuthCacheClient *redis.Client
)

// InitCache initializes the generic Redis cache client (using DB from AppConfig for general caching).
func InitCache() {
	CacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisCacheDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := CacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Cache): %v", err)
	}
}

// GetCacheClient returns the generic cache client.
func GetCacheClient() *redis.Client {
	if CacheClient == nil {
		InitCache()
	}
	return CacheClient
}

// InitAuthCache initializes the Redis client for authorization caching (using DB from AppConfig for auth cache).
func InitAuthCache() {
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
}

// GetAuthCacheClient returns the Redis client for authorization caching.
func GetAuthCacheClient() *redis.Client {
	if AuthCacheClient == nil {
		InitAuthCache()
	}
	return AuthCacheClient
}
