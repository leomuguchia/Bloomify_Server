package schedulerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UpdateEmbeddedTimeSlotAggregates updates an embedded timeslot within the provider document.
// It increments the appropriate booked units counter (standard or priority) and increments the version.
func (repo *MongoSchedulerRepo) UpdateEmbeddedTimeSlotAggregates(providerID string, slot models.TimeSlot, date string, units int, priority bool, currentVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      slot.ID,
				"date":    date,
				"version": currentVersion,
			},
		},
	}
	var update bson.M
	if priority {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$.bookedUnitsPriority": units,
				"timeSlots.$.version":             1,
			},
		}
	} else {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$.bookedUnitsStandard": units,
				"timeSlots.$.version":             1,
			},
		}
	}
	res, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update embedded timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("embedded timeslot aggregate update failed (version mismatch or missing document)")
	}
	return nil
}

// SetEmbeddedTimeSlotBlockReason sets the block reason (as a string) for an embedded timeslot within the provider document.
// An empty string means the slot is not blocked.
func (repo *MongoSchedulerRepo) SetEmbeddedTimeSlotBlockReason(providerID string, slot models.TimeSlot, date, blockReason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":   slot.ID,
				"date": date,
			},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"timeSlots.$.blockReason": blockReason,
		},
	}
	_, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update block reason for embedded timeslot: %w", err)
	}
	return nil
}

// RollbackEmbeddedTimeSlotAggregates decrements the booked units in the embedded timeslot,
// used when a booking must be cancelled.
func (repo *MongoSchedulerRepo) RollbackEmbeddedTimeSlotAggregates(providerID string, slotID string, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      slotID,
				"date":    date,
				"version": bson.M{"$gt": expectedVersion},
			},
		},
	}
	var update bson.M
	if isPriority {
		update = bson.M{"$inc": bson.M{"timeSlots.$.bookedUnitsPriority": -units}}
	} else {
		update = bson.M{"$inc": bson.M{"timeSlots.$.bookedUnitsStandard": -units}}
	}
	res, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to rollback embedded timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("rollback update failed; timeslot not found or version mismatch")
	}
	return nil
}

// SumOverlappingBookings aggregates the total booked units (both standard and priority)
// for a provider on a given date and time range.
func (repo *MongoSchedulerRepo) SumOverlappingBookings(providerID, date string, start, end int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"providerId": providerID,
			"date":       date,
			"start":      bson.M{"$lt": end},
			"end":        bson.M{"$gt": start},
		}}},
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

// SumOverlappingBookingsForStandard aggregates booked units for non-priority bookings.
func (repo *MongoSchedulerRepo) SumOverlappingBookingsForStandard(providerID, date string, start, end int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"providerId": providerID,
			"date":       date,
			"start":      bson.M{"$lt": end},
			"end":        bson.M{"$gt": start},
			"priority":   false,
		}}},
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

// SumOverlappingBookingsForPriority aggregates booked units for priority bookings.
func (repo *MongoSchedulerRepo) SumOverlappingBookingsForPriority(providerID, date string, start, end int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"providerId": providerID,
			"date":       date,
			"start":      bson.M{"$lt": end},
			"end":        bson.M{"$gt": start},
			"priority":   true,
		}}},
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
