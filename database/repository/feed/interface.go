package feedRepo

import (
	"bloomify/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FeedRepository interface {
	UpsertFeedBlock(ctx context.Context, block *models.FeedBlock) error
	GetFeedBlockByTheme(ctx context.Context, theme string) (*models.FeedBlock, error)
	GetAllFeedBlocks(ctx context.Context, limit, offset int) ([]models.FeedBlock, error)
	DeleteFeedBlock(ctx context.Context, theme string) error
}

type MongoFeedRepo struct {
	coll *mongo.Collection
}

func NewMongoFeedRepo(db *mongo.Database) FeedRepository {
	return &MongoFeedRepo{
		coll: db.Collection("feed_blocks"),
	}
}

func (r *MongoFeedRepo) UpsertFeedBlock(ctx context.Context, block *models.FeedBlock) error {
	if block.Theme == "" {
		return errors.New("feed block must have a theme")
	}
	filter := bson.M{"theme": block.Theme}
	update := bson.M{"$set": block}
	opts := options.Update().SetUpsert(true)

	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoFeedRepo) GetFeedBlockByTheme(ctx context.Context, theme string) (*models.FeedBlock, error) {
	var block models.FeedBlock
	err := r.coll.FindOne(ctx, bson.M{"theme": theme}).Decode(&block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (r *MongoFeedRepo) GetAllFeedBlocks(ctx context.Context, limit, offset int) ([]models.FeedBlock, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{"theme", 1}})
	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blocks []models.FeedBlock
	if err := cursor.All(ctx, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func (r *MongoFeedRepo) DeleteFeedBlock(ctx context.Context, theme string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"theme": theme})
	return err
}
