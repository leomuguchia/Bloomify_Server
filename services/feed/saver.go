package feed

import (
	feedRepo "bloomify/database/repository/feed"
	"bloomify/models"
	"bloomify/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var repo feedRepo.FeedRepository

func InitFeed(repoInstance feedRepo.FeedRepository) {
	repo = repoInstance
}

func SaveFeedBlock(ctx context.Context, key string, block *models.FeedBlock) error {
	// Save to MongoDB using the repository
	if err := repo.UpsertFeedBlock(ctx, block); err != nil {
		return err
	}

	// Save to Redis
	if err := saveToRedis(ctx, key, block); err != nil {
		log.Printf("warning: failed to save to Redis: %v", err)
	}
	return nil
}

func saveToRedis(ctx context.Context, key string, block *models.FeedBlock) error {
	client := utils.GetFeedCacheClient()
	redisKey := fmt.Sprintf("feed:aggregates:%s", key)
	bytes, err := json.Marshal(block)
	if err != nil {
		return err
	}
	return client.Set(ctx, redisKey, bytes, 2*time.Hour).Err()
}
