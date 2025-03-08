package providerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}
	if criteria.ServiceType != "" {
		filter["service_type"] = bson.M{"$regex": criteria.ServiceType, "$options": "i"}
	}
	if criteria.Location != "" {
		filter["location"] = bson.M{"$regex": criteria.Location, "$options": "i"}
	}
	if criteria.MaxDistanceKm > 0 {
		maxDistanceMeters := criteria.MaxDistanceKm * 1000
		filter["location_geo"] = bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{criteria.Longitude, criteria.Latitude},
				},
				"$maxDistance": maxDistanceMeters,
			},
		}
	}
	filter["status"] = bson.M{"$in": []string{"active", "online"}}

	opts := options.Find()

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("advanced search query failed: %w", err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	for cursor.Next(ctx) {
		var p models.Provider
		if err := cursor.Decode(&p); err != nil {
			return nil, fmt.Errorf("failed to decode provider: %w", err)
		}
		providers = append(providers, p)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found matching criteria")
	}

	return providers, nil
}
