package schedulerRepo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (repo *MongoSchedulerRepo) SumOverlappingBookings(providerID, date string, start, end int, priorityFilter *bool) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	match := bson.M{
		"providerId": providerID,
		"date":       date,
		"start":      bson.M{"$lt": end},
		"end":        bson.M{"$gt": start},
	}

	// Add priority filter only if explicitly provided
	if priorityFilter != nil {
		match["priority"] = *priorityFilter
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$units"},
		}}},
	}

	cursor, err := repo.bookingColl.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("aggregation error: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, fmt.Errorf("error decoding aggregation result: %w", err)
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0].Total, nil
}
