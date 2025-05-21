package recordsRepo

import (
	"bloomify/database"
	"bloomify/models"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type HistoricalRecordRepository interface {
	Create(ctx context.Context, record models.HistoricalRecord) (string, error)
	GetByID(ctx context.Context, id string) (*models.HistoricalRecord, error)
	GetByProviderID(ctx context.Context, providerID string) ([]models.HistoricalRecord, error)
	DeleteByID(ctx context.Context, id string) error
}

type mongoRecordRepo struct {
	coll *mongo.Collection
}

// NewMongoRecordRepo returns a new HistoricalRecordRepository instance using MongoDB.
func NewMongoRecordRepo() HistoricalRecordRepository {
	db := database.MongoClient.Database("bloomify")
	return &mongoRecordRepo{
		coll: db.Collection("historical_records"),
	}
}
