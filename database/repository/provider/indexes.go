package providerRepo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *MongoProviderRepo) ensureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	base := []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "profile.email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "serviceCatalogue.service.id", Value: 1}}},
		{Keys: bson.D{{Key: "serviceCatalogue.customOptions.option", Value: 1}}},
		{Keys: bson.D{{Key: "serviceCatalogue.mode", Value: 1}}},
		{Keys: bson.D{{Key: "profile.status", Value: 1}}},
		// Single 2dsphere index – required for geo queries
		{Keys: bson.D{{Key: "profile.locationGeo", Value: "2dsphere"}}},
	}

	// Partial index for timeSlotRefs (for faster filtering)
	partialOpts := options.Index().SetPartialFilterExpression(bson.M{
		"timeSlotRefs.0": bson.M{"$exists": true},
	})
	timeslotIdx := mongo.IndexModel{
		Keys:    bson.D{{Key: "timeSlotRefs", Value: 1}},
		Options: partialOpts,
	}

	indexModels := append(base, timeslotIdx)
	if _, err := r.coll.Indexes().CreateMany(ctx, indexModels); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}
