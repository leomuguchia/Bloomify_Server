package schedulerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetAvailableTimeSlots fetches timeslots for a provider for the given date using an aggregation pipeline.
// If no provider is found, it returns an empty slice.
func (repo *MongoSchedulerRepo) GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"id": providerID}}},
		{{Key: "$project", Value: bson.M{
			"timeSlots": bson.M{
				"$filter": bson.M{
					"input": "$timeSlots",
					"as":    "ts",
					"cond":  bson.M{"$eq": []interface{}{"$$ts.date", date}},
				},
			},
		}}},
	}

	cursor, err := repo.providerColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregating provider with id %s: %w", providerID, err)
	}
	defer cursor.Close(ctx)

	// Temporary structure for the aggregation result.
	var results []struct {
		TimeSlots []models.TimeSlot `bson:"timeSlots"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("error decoding aggregation results: %w", err)
	}

	// If no provider is found, simply return an empty slice.
	if len(results) == 0 {
		return []models.TimeSlot{}, nil
	}

	return results[0].TimeSlots, nil
}

// GetMaxAvailableDate retrieves the maximum timeslot date for a provider,
// considering only timeslots from today onward.
func (repo *MongoSchedulerRepo) GetMaxAvailableDate(providerID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"id": providerID}}},
		{{Key: "$project", Value: bson.M{
			"timeSlots": bson.M{
				"$filter": bson.M{
					"input": "$timeSlots",
					"as":    "ts",
					"cond": bson.M{"$gte": []interface{}{
						bson.M{"$toDate": "$$ts.date"},
						time.Now(),
					}},
				},
			},
		}}},
		{{Key: "$unwind", Value: "$timeSlots"}},
		{{Key: "$group", Value: bson.M{
			"_id":              nil,
			"maxAvailableDate": bson.M{"$max": bson.M{"$toDate": "$timeSlots.date"}},
		}}},
		{{Key: "$project", Value: bson.M{
			"maxAvailableDate": bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$maxAvailableDate"}},
			"_id":              0,
		}}},
	}

	cursor, err := repo.providerColl.Aggregate(ctx, pipeline)
	if err != nil {
		return "", fmt.Errorf("error aggregating max date for provider %s: %w", providerID, err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		MaxAvailableDate string `bson:"maxAvailableDate"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return "", fmt.Errorf("error decoding max date: %w", err)
	}

	if len(results) == 0 || results[0].MaxAvailableDate == "" {
		return "", nil
	}
	return results[0].MaxAvailableDate, nil
}
