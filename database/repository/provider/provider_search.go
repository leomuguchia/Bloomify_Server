package providerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	logger := utils.GetLogger()
	logger.Debug("AdvancedSearch: received criteria", zap.Any("criteria", criteria))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pipeline mongo.Pipeline

	// 1. Geo filter
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

	// 2. Filter by status and at least one timeslot
	match := bson.M{
		"profile.status": bson.M{"$in": []string{"active", "online"}},
		// ensure timeSlotRefs exists and is non-empty
		"$expr": bson.M{"$gt": bson.A{
			bson.M{"$size": bson.M{"$ifNull": bson.A{"$timeSlotRefs", bson.A{}}}},
			0,
		}},
	}
	if criteria.ServiceType != "" {
		match["serviceCatalogue.service.id"] = bson.M{"$regex": criteria.ServiceType, "$options": "i"}
	}
	if criteria.CustomOption != "" {
		match["serviceCatalogue.customOptions"] = bson.M{
			"$elemMatch": bson.M{"option": bson.M{"$regex": criteria.CustomOption, "$options": "i"}},
		}
	}

	if len(criteria.Modes) > 0 {
		match["serviceCatalogue.mode"] = bson.M{
			"$in": criteria.Modes,
		}
	}

	pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})

	// 3. Add computed fields, safely handling missing arrays
	pipeline = append(pipeline, bson.D{{
		Key: "$addFields", Value: bson.M{
			"slotCount": bson.M{
				"$size": bson.M{"$ifNull": bson.A{"$timeSlotRefs", bson.A{}}},
			},
			"activeCount": bson.M{
				"$size": bson.M{"$ifNull": bson.A{"$activeBookings", bson.A{}}},
			},
		},
	}})

	// 🚫 4. No $sort here — we sort in Go

	logger.Debug("AdvancedSearch: final pipeline", zap.Any("pipeline", pipeline))

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		logger.Error("AdvancedSearch: aggregation failed", zap.Error(err))
		return nil, fmt.Errorf("aggregation query failed: %w", err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	if err := cursor.All(ctx, &providers); err != nil {
		logger.Error("AdvancedSearch: decode failed", zap.Error(err))
		return nil, fmt.Errorf("failed to decode providers: %w", err)
	}

	return providers, nil
}
