package utils

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var CacheClient *redis.Client

func InitCache() {
	CacheClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Adjust as needed
		Password: "",
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := CacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

func GetCacheClient() *redis.Client {
	if CacheClient == nil {
		InitCache()
	}
	return CacheClient
}
