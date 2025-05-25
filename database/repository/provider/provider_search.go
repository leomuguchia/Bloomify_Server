package providerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pipeline mongo.Pipeline

	// 1) $geoNear: must come first to filter+sort by distance
	if criteria.MaxDistanceKm > 0 && len(criteria.LocationGeo.Coordinates) == 2 {
		pipeline = append(pipeline, bson.D{
			{Key: "$geoNear", Value: bson.D{
				{Key: "near", Value: bson.D{
					{Key: "type", Value: "Point"},
					{Key: "coordinates", Value: criteria.LocationGeo.Coordinates},
				}},
				{Key: "distanceField", Value: "distance"},
				{Key: "spherical", Value: true},
				{Key: "maxDistance", Value: criteria.MaxDistanceKm * 1000},
			}},
		})
	}

	// 2) $match: active/online + must have at least one timeslot
	matchFilter := bson.M{
		"profile.status": bson.M{"$in": []string{"active", "online"}},
		"timeSlotRefs":   bson.M{"$exists": true, "$ne": bson.A{}},
	}
	if criteria.ServiceType != "" {
		matchFilter["serviceCatalogue.service.id"] = bson.M{"$regex": criteria.ServiceType, "$options": "i"}
	}
	if criteria.CustomOption != "" {
		matchFilter["serviceCatalogue.customOptions"] = bson.M{"$elemMatch": bson.M{
			"option": bson.M{"$regex": criteria.CustomOption, "$options": "i"},
		}}
	}
	if criteria.Mode != "" {
		matchFilter["serviceCatalogue.mode"] = criteria.Mode
	}
	pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchFilter}})

	// 3) $addFields: compute slotCount and activeCount
	pipeline = append(pipeline, bson.D{
		{Key: "$addFields", Value: bson.M{
			"slotCount":   bson.M{"$size": "$timeSlotRefs"},
			"activeCount": bson.M{"$size": "$activeBookings"},
		}},
	})

	// 4) $sort: verified first, then most slots, nearest, then lightest load
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
		{Key: "profile.advancedVerified", Value: -1}, // true before false
		{Key: "slotCount", Value: -1},                // more slots first
		{Key: "distance", Value: 1},                  // nearer first
		{Key: "activeCount", Value: 1},               // fewer active bookings first
	}}})

	// Execute pipeline
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation query failed: %w", err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	if err := cursor.All(ctx, &providers); err != nil {
		return nil, fmt.Errorf("failed to decode providers: %w", err)
	}
	return providers, nil
}
