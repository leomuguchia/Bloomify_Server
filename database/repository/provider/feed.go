package providerRepo

import (
	"bloomify/models"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *MongoProviderRepo) FetchTopProviders(ctx context.Context, page, limit int) ([]models.Provider, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"serviceCatalogue.service.name": bson.M{"$exists": true}}}},
		{{Key: "$sort", Value: bson.D{{Key: "completedBookings", Value: -1}, {Key: "profile.rating", Value: -1}}}},
		{{Key: "$skip", Value: int64(page * limit)}},
		{{Key: "$limit", Value: int64(limit)}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.Provider
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
