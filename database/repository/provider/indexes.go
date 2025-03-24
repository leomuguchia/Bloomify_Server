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

	indexModels := []mongo.IndexModel{
		// Unique index on provider's "id".
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Unique index on the provider's email stored in "profile.email".
		{Keys: bson.D{{Key: "profile.email", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Index on the service type in the service catalogue.
		{Keys: bson.D{{Key: "serviceCatalogue.serviceType", Value: 1}}},
		// Create a 2dsphere index on "locationGeo" for geospatial queries.
		{Keys: bson.D{{Key: "profile.locationGeo", Value: "2dsphere"}}},
		// Optionally, index the provider status in the profile.
		{Keys: bson.D{{Key: "profile.status", Value: 1}}},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}
