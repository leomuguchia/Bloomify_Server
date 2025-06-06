// File: services/feed/utils/cache.go
package feedUtils

import (
	"bloomify/models"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type FeedCache interface {
	SetBlock(ctx context.Context, block models.FeedBlock) error
	GetAllBlocks(ctx context.Context) ([]models.FeedBlock, error)
	DeleteBlock(ctx context.Context, id string) error
}

type RedisFeedCache struct {
	client *redis.Client
}

func NewRedisFeedCache(client *redis.Client) FeedCache {
	return &RedisFeedCache{client: client}
}

const cacheKeyPrefix = "feed:aggregates:"

func blockKey(id string) string {
	return fmt.Sprintf("%s%s", cacheKeyPrefix, id)
}

func (c *RedisFeedCache) SetBlock(ctx context.Context, block models.FeedBlock) error {
	data, err := json.Marshal(block)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, blockKey(block.ID), data, 24*time.Hour).Err() // example TTL
}

func (c *RedisFeedCache) GetAllBlocks(ctx context.Context) ([]models.FeedBlock, error) {
	keys, err := c.client.Keys(ctx, cacheKeyPrefix+"*").Result()
	if err != nil {
		return nil, err
	}

	var blocks []models.FeedBlock
	for _, key := range keys {
		val, err := c.client.Get(ctx, key).Result()
		if err != nil {
			continue // skip corrupt/missing
		}

		var block models.FeedBlock
		if err := json.Unmarshal([]byte(val), &block); err != nil {
			continue
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (c *RedisFeedCache) DeleteBlock(ctx context.Context, id string) error {
	return c.client.Del(ctx, blockKey(id)).Err()
}
