package recordsRepo

import (
	"bloomify/models"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// Create inserts a new historical record and returns its ID.
func (r *mongoRecordRepo) Create(ctx context.Context, record models.HistoricalRecord) (string, error) {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()

	_, err := r.coll.InsertOne(ctx, record)
	if err != nil {
		return "", err
	}
	return record.ID, nil
}

// GetByID returns a historical record by its ID.
func (r *mongoRecordRepo) GetByID(ctx context.Context, id string) (*models.HistoricalRecord, error) {
	var record models.HistoricalRecord
	err := r.coll.FindOne(ctx, bson.M{"id": id}).Decode(&record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByProviderID fetches all records associated with a specific provider.
func (r *mongoRecordRepo) GetByProviderID(ctx context.Context, providerID string) ([]models.HistoricalRecord, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"providerId": providerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.HistoricalRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// DeleteByID removes a historical record by ID.
func (r *mongoRecordRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("record not found")
	}
	return nil
}
