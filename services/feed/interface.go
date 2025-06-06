package feed

import (
	"bloomify/models"
	"context"
)

type FeedCache interface {
	CacheBlock(ctx context.Context, block models.FeedBlock) error
	GetCachedBlocks(ctx context.Context) ([]models.FeedBlock, error)
	DeleteCachedBlock(ctx context.Context, id string) error
}
