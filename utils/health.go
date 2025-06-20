package utils

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
)

// HealthStatus represents current status of external services.
type HealthStatus struct {
	Mongo     bool      `json:"mongo"`
	Redis     []bool    `json:"redis"`
	CheckedAt time.Time `json:"checkedAt"`
}

var (
	currentHealth HealthStatus
	mu            sync.RWMutex
)

// GetHealthStatus returns latest stored health snapshot.
func GetHealthStatus() HealthStatus {
	mu.RLock()
	defer mu.RUnlock()
	return currentHealth
}

// StartHealthMonitor performs periodic health checks and updates in-memory state.
func StartHealthMonitor(redisClients []*redis.Client, mongoClient *mongo.Client) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		ctx := context.Background()

		for range ticker.C {
			var redisHealth []bool

			for _, client := range redisClients {
				err := client.Ping(ctx).Err()
				redisHealth = append(redisHealth, err == nil)
			}

			mongoHealthy := mongoClient.Ping(ctx, nil) == nil

			mu.Lock()
			currentHealth = HealthStatus{
				Mongo:     mongoHealthy,
				Redis:     redisHealth,
				CheckedAt: time.Now(),
			}
			mu.Unlock()
		}
	}()
}
