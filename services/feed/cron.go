package feed

import (
	providerRepo "bloomify/database/repository/provider"
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

func StartFeedCron(ctx context.Context, pr providerRepo.ProviderRepository, redisClient *redis.Client) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Feed cron shutdown signal received.")
			return
		case <-ticker.C:
			log.Println("Running hourly Discover Engine")
			if err := feed.RunFeedAggregation(ctx, pr); err != nil {
				log.Printf("Feed aggregation failed: %v\n", err)
			}
		}
	}
}
