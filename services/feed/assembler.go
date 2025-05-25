package feed

import (
	"bloomify/models"
	"fmt"
	"strings"
)

func AssembleFeedBlocksFromProviders(providers []models.Provider) map[string]*models.FeedBlock {
	blocks := make(map[string]*models.FeedBlock)

	for _, provider := range providers {
		catalog := provider.ServiceCatalogue
		service := catalog.Service
		options := catalog.CustomOptions
		intent := classifyIntent(service, options)

		for _, opt := range options {
			item := models.FeedItem{
				Title:        fmt.Sprintf("%s %s", strings.Title(opt.Option), strings.Title(service.ID)),
				CustomOption: opt.Option,
				ServiceType:  service.ID,
				Description:  fmt.Sprintf("%s version of %s services", opt.Option, service.ID),
				Rating:       provider.Profile.Rating,
			}

			blockKey := fmt.Sprintf("%s_%s", intent, service.ID)
			if _, exists := blocks[blockKey]; !exists {
				blocks[blockKey] = &models.FeedBlock{
					Theme:     fmt.Sprintf("%s: %s", strings.Title(intent), strings.Title(service.ID)),
					Tags:      []string{intent, service.ID},
					FeedItems: []models.FeedItem{},
				}
			}
			blocks[blockKey].FeedItems = append(blocks[blockKey].FeedItems, item)
		}
	}

	return blocks
}

// classifyIntent determines the intent based on the service and options.
func classifyIntent(service models.Service, options []models.CustomOption) string {
	// TODO: Implement actual intent classification logic.
	// For now, return a placeholder or use a field from service/options.
	return "default"
}
