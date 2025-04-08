package schedulerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// UpdateTimeSlotAggregates updates the aggregate counters (booked units) in an embedded timeslot.
// This method uses an update with an array filter on the provider document.
func (repo *MongoSchedulerRepo) UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      ts.ID,
				"date":    date,
				"version": expectedVersion,
			},
		},
	}
	update := bson.M{
		"$inc": bson.M{},
	}
	if isPriority {
		update["$inc"].(bson.M)["timeSlots.$.bookedUnitsPriority"] = units
	} else {
		update["$inc"].(bson.M)["timeSlots.$.bookedUnitsStandard"] = units
	}
	// Increment the version as well.
	update["$inc"].(bson.M)["timeSlots.$.version"] = 1

	res, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("timeslot aggregate update failed (version mismatch or missing document)")
	}
	return nil
}

// RollbackTimeSlotAggregates decrements the aggregate counters in an embedded timeslot, used when a booking is cancelled.
func (repo *MongoSchedulerRepo) RollbackTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      ts.ID,
				"date":    date,
				"version": bson.M{"$gt": expectedVersion},
			},
		},
	}
	update := bson.M{
		"$inc": bson.M{},
	}
	if isPriority {
		update["$inc"].(bson.M)["timeSlots.$.bookedUnitsPriority"] = -units
	} else {
		update["$inc"].(bson.M)["timeSlots.$.bookedUnitsStandard"] = -units
	}

	res, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to rollback timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("rollback update failed; timeslot not found or version mismatch")
	}
	return nil
}
