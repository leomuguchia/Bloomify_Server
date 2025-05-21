// File: database/repository/timeslot/queries.go
package timeslotRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (repo *mongoTimeSlotRepo) GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"providerId": providerID,
		"date":       date,
		"blocked":    false,
	}

	cursor, err := repo.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeslots: %w", err)
	}
	defer cursor.Close(ctx)

	var slots []models.TimeSlot
	if err := cursor.All(ctx, &slots); err != nil {
		return nil, fmt.Errorf("error decoding timeslots: %w", err)
	}

	return slots, nil
}

func (repo *mongoTimeSlotRepo) GetMaxAvailableDate(providerID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"providerId": providerID,
			"date":       bson.M{"$gte": time.Now().Format("2006-01-02")},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":     nil,
			"maxDate": bson.M{"$max": "$date"},
		}}},
	}

	cursor, err := repo.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return "", fmt.Errorf("failed to aggregate max date: %w", err)
	}
	defer cursor.Close(ctx)

	var result []struct {
		MaxDate string `bson:"maxDate"`
	}
	if err := cursor.All(ctx, &result); err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}

	if len(result) == 0 {
		return "", nil
	}
	return result[0].MaxDate, nil
}

func (repo *mongoTimeSlotRepo) GetTimeSlotByID(providerID, slotID, date string, start, end int) (*models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id":         slotID,
		"providerId": providerID,
		"date":       date,
		"start":      start,
		"end":        end,
	}

	var slot models.TimeSlot
	err := repo.coll.FindOne(ctx, filter).Decode(&slot)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("slot not found")
		}
		return nil, fmt.Errorf("find error: %w", err)
	}

	return &slot, nil
}
