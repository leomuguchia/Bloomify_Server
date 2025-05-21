// FILE: database/repository/timeslot/indexes.go
package timeslotRepo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes creates the necessary indexes on the timeslots collection.
func (r *mongoTimeSlotRepo) EnsureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModels := []mongo.IndexModel{
		// Unique index on TimeSlot ID
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_id"),
		},
		// Compound index for providerId and date (primary query pattern)
		{
			Keys:    bson.D{{Key: "providerId", Value: 1}, {Key: "date", Value: 1}},
			Options: options.Index().SetName("provider_date_idx"),
		},
		// Compound index for providerId + date + blocked for filtering blocked slots
		{
			Keys:    bson.D{{Key: "providerId", Value: 1}, {Key: "date", Value: 1}, {Key: "blocked", Value: 1}},
			Options: options.Index().SetName("provider_date_blocked_idx"),
		},
		{
			Keys:    bson.D{{Key: "providerId", Value: 1}, {Key: "date", Value: 1}, {Key: "start", Value: 1}, {Key: "end", Value: 1}},
			Options: options.Index().SetName("provider_date_start_end_idx"),
		},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create timeslot indexes: %w", err)
	}
	return nil
}
