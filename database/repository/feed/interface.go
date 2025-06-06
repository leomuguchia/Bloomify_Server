package feedRepo

import (
	"bloomify/models"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FeedRepository interface {
	SaveBlock(ctx context.Context, block models.FeedBlock) error
	LoadAllBlocks(ctx context.Context) ([]models.FeedBlock, error)
	IncrementAccess(ctx context.Context, id string) error
	DeleteBlock(ctx context.Context, id string) error
}

type MongoFeedRepo struct {
	coll *mongo.Collection
}

func NewMongoFeedRepo(db *mongo.Database) FeedRepository {
	return &MongoFeedRepo{
		coll: db.Collection("feed"),
	}
}

func (m *MongoFeedRepo) SaveBlock(ctx context.Context, block models.FeedBlock) error {
	block.CreatedAt = time.Now()
	_, err := m.coll.InsertOne(ctx, block)
	return err
}

func (m *MongoFeedRepo) LoadAllBlocks(ctx context.Context) ([]models.FeedBlock, error) {
	cursor, err := m.coll.Find(ctx, bson.M{})
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

func (m *MongoFeedRepo) IncrementAccess(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid feed block id")
	}
	_, err = m.coll.UpdateByID(ctx, objID, bson.M{
		"$inc": bson.M{"accessCount": 1},
	})
	return err
}

func (m *MongoFeedRepo) DeleteBlock(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid feed block id")
	}
	_, err = m.coll.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}
