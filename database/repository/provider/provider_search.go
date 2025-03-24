package providerRepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	logger := utils.GetLogger()
	logger.Debug("AdvancedSearch: received criteria", zap.Any("criteria", criteria))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}

	// Filter by baseline service type within the service catalogue.
	if criteria.ServiceType != "" {
		filter["serviceCatalogue.serviceType"] = bson.M{
			"$regex":   criteria.ServiceType,
			"$options": "i",
		}
	}

	// Filter by service mode within the service catalogue.
	if criteria.ServiceMode != "" {
		filter["serviceCatalogue.mode"] = bson.M{
			"$regex":   criteria.ServiceMode,
			"$options": "i",
		}
	}

	// Adjusted filter: use provider's profile for geo-location.
	if criteria.MaxDistanceKm > 0 {
		maxDistanceMeters := criteria.MaxDistanceKm * 1000
		filter["profile.locationGeo"] = bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": criteria.LocationGeo.Coordinates,
				},
				"$maxDistance": maxDistanceMeters,
			},
		}
	}

	// Ensure provider status is active or online.
	filter["profile.status"] = bson.M{"$in": []string{"active", "online"}}

	logger.Debug("AdvancedSearch: constructed filter", zap.Any("filter", filter))

	opts := options.Find()

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		logger.Error("AdvancedSearch: query failed", zap.Error(err))
		return nil, fmt.Errorf("advanced search query failed: %w", err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	for cursor.Next(ctx) {
		var p models.Provider
		if err := cursor.Decode(&p); err != nil {
			logger.Error("AdvancedSearch: failed to decode provider", zap.Error(err))
			return nil, fmt.Errorf("failed to decode provider: %w", err)
		}
		providers = append(providers, p)
	}
	if err := cursor.Err(); err != nil {
		logger.Error("AdvancedSearch: cursor error", zap.Error(err))
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if len(providers) == 0 {
		errMsg := "no providers found matching criteria"
		logger.Warn("AdvancedSearch: no providers found", zap.Any("filter", filter))
		return nil, errors.New(errMsg)
	}

	logger.Debug("AdvancedSearch: found providers", zap.Int("count", len(providers)))
	return providers, nil
}
