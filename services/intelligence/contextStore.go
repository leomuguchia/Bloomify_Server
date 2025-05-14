// File: service/ai/contextStore.go
package ai

import (
	"context"
	"encoding/json"
	"time"

	"bloomify/models"

	"github.com/go-redis/redis/v8"
)

const aiContextPrefix = "ai:ctx:"

type RedisContextStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisContextStore(client *redis.Client, ttl time.Duration) *RedisContextStore {
	return &RedisContextStore{client: client, ttl: ttl}
}

func (s *RedisContextStore) Get(ctx context.Context, userID string) (*models.AIContext, error) {
	key := aiContextPrefix + userID
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return &models.AIContext{}, nil
	}
	if err != nil {
		return nil, err
	}
	var aiCtx models.AIContext
	if err := json.Unmarshal([]byte(data), &aiCtx); err != nil {
		return nil, err
	}
	return &aiCtx, nil
}

func (s *RedisContextStore) Set(ctx context.Context, userID string, aiCtx *models.AIContext) error {
	key := aiContextPrefix + userID
	b, err := json.Marshal(aiCtx)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, b, s.ttl).Err()
}

func (s *RedisContextStore) Clear(ctx context.Context, userID string) error {
	key := aiContextPrefix + userID
	return s.client.Del(ctx, key).Err()
}
