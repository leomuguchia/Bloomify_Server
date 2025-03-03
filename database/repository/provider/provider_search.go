package providerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AdvancedSearch performs an advanced search based on the provided criteria.
// It returns at most 20 providers. Only providers with status "active" or "online"
// and non-zero latitude/longitude are returned.
func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build the filter document based on criteria.
	filter := bson.M{}
	if criteria.ServiceType != "" {
		filter["service_type"] = bson.M{"$regex": criteria.ServiceType, "$options": "i"}
	}
	if criteria.Location != "" {
		filter["location"] = bson.M{"$regex": criteria.Location, "$options": "i"}
	}
	if criteria.MinRating > 0 {
		filter["rating"] = bson.M{"$gte": criteria.MinRating}
	}
	if criteria.MinCompletedBookings > 0 {
		filter["completed_bookings"] = bson.M{"$gte": criteria.MinCompletedBookings}
	}
	// For geospatial search, assume documents have a GeoJSON field "location_geo"
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
	// Ensure provider is active/online and has a valid location.
	filter["status"] = bson.M{"$in": []string{"active", "online"}}
	filter["latitude"] = bson.M{"$ne": 0}
	filter["longitude"] = bson.M{"$ne": 0}

	// Set options: sort by rating (descending) and limit to 20 results.
	opts := options.Find().
		SetSort(bson.D{{Key: "rating", Value: -1}}).
		SetLimit(20)

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
