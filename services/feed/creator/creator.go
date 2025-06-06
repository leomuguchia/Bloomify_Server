package feedCreator

import (
	feedRepo "bloomify/database/repository/feed"
	"bloomify/models"
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type Creator struct {
	feedRepo        feedRepo.FeedRepository
	redis           *redis.Client
	providerFetcher ProviderFetcher
}

type ProviderFetcher interface {
	FetchProvidersBatch(ctx context.Context, offset, limit int) ([]models.Provider, error)
}

func NewCreator(feedRepo feedRepo.FeedRepository, redis *redis.Client, fetcher ProviderFetcher) *Creator {
	return &Creator{
		feedRepo:        feedRepo,
		redis:           redis,
		providerFetcher: fetcher,
	}
}

var feedCacheKeyPrefix = "feed:aggregates:"
var feedCacheTTL = 24 * time.Hour

func (c *Creator) Run(ctx context.Context) {
	log.Println("[FeedCreator] Starting block generation...")

	const batchSize = 100
	offset := 0

	for {
		providers, err := c.providerFetcher.FetchProvidersBatch(ctx, offset, batchSize)
		if err != nil {
			log.Printf("[FeedCreator] Failed to fetch provider batch: %v", err)
			break
		}
		if len(providers) == 0 {
			log.Println("[FeedCreator] All providers processed.")
			break
		}

		blocks := ProcessProvidersIntoFeedBlocks(providers)

		for _, block := range blocks {
			err := c.feedRepo.SaveBlock(ctx, block)
			if err != nil {
				log.Printf("[FeedCreator] Failed to save block to DB: %v", err)
			}
			cacheKey := feedCacheKeyPrefix + block.Theme
			err = c.redis.Set(ctx, cacheKey, block, time.Hour*6).Err()
			if err != nil {
				log.Printf("[FeedCreator] Failed to cache block in Redis: %v", err)
			}
		}

		offset += batchSize
	}
}
