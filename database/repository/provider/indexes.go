package providerRepo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ensureIndexes creates indexes for frequently used fields in queries.
func (r *MongoProviderRepo) ensureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slotCountIdx := mongo.IndexModel{
		Keys: bson.D{{Key: "slotCount", Value: 1}},
	}
	activeCountIdx := mongo.IndexModel{
		Keys: bson.D{{Key: "activeCount", Value: 1}},
	}
	// Partial index: only providers with at least one timeslot
	partialOpts := options.Index().SetPartialFilterExpression(bson.M{
		"timeSlotRefs.0": bson.M{"$exists": true},
	})
	timeslotIdx := mongo.IndexModel{
		Keys:    bson.D{{Key: "timeSlotRefs", Value: 1}},
		Options: partialOpts,
	}
	// Compound geo + verified + slotCount
	geoCompoundIdx := mongo.IndexModel{
		Keys: bson.D{
			{Key: "profile.locationGeo", Value: "2dsphere"},
			{Key: "profile.advancedVerified", Value: -1},
			{Key: "slotCount", Value: -1},
		},
	}

	base := []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "profile.email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "serviceCatalogue.service.id", Value: 1}}},
		{Keys: bson.D{{Key: "serviceCatalogue.customOptions.option", Value: 1}}},
		{Keys: bson.D{{Key: "serviceCatalogue.mode", Value: 1}}},
		{Keys: bson.D{{Key: "profile.status", Value: 1}}},
		// keep your simple 2dsphere in case you query geo alone
		{Keys: bson.D{{Key: "profile.locationGeo", Value: "2dsphere"}}},
	}

	// Combine all indexes
	indexModels := append(base, slotCountIdx, activeCountIdx, timeslotIdx, geoCompoundIdx)
	if _, err := r.coll.Indexes().CreateMany(ctx, indexModels); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}
