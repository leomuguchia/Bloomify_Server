package feed

import (
	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"
	"context"
	"log"
	"sync"
)

const (
	pageSize = 50
	maxPages = 10
)

func RunFeedAggregation(ctx context.Context, repo providerRepo.ProviderRepository) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	for page := 0; page < maxPages; page++ {
		providers, err := repo.FetchTopProviders(ctx, page, pageSize)
		if err != nil {
			return err
		}
		if len(providers) == 0 {
			break
		}

		wg.Add(1)
		go func(batch []models.Provider) {
			defer wg.Done()
			blocks := AssembleFeedBlocksFromProviders(batch)
			for key, block := range blocks {
				if err := SaveFeedBlock(ctx, key, block); err != nil {
					log.Printf("failed to save block %s: %v", key, err)
				}
			}
		}(providers)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(errChan)
	}()

	return <-errChan
}
