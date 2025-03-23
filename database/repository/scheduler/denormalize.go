package schedulerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpdateTimeSlotAggregates atomically increments booking aggregates for a timeslot.
// It updates the provider's timeSlots array element that matches the given criteria.
func (repo *MongoSchedulerRepo) UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": providerID}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{
				"elem.start":   ts.Start,
				"elem.end":     ts.End,
				"elem.date":    date,
				"elem.version": expectedVersion,
			},
		},
	})

	var update bson.M
	if isPriority {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$[elem].bookedUnitsPriority": units,
				"timeSlots.$[elem].version":             1,
			},
		}
	} else {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$[elem].bookedUnitsStandard": units,
				"timeSlots.$[elem].version":             1,
			},
		}
	}

	res, err := repo.providerColl.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update aggregates error: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("update aggregates failed: no matching document found or version mismatch")
	}
	return nil
}

// RollbackTimeSlotAggregates atomically decrements booking aggregates for a timeslot.
// This is used when a booking is cancelled (e.g., due to payment failure).
func (repo *MongoSchedulerRepo) RollbackTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": providerID}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{
				"elem.start":   ts.Start,
				"elem.end":     ts.End,
				"elem.date":    date,
				"elem.version": expectedVersion,
			},
		},
	})

	var update bson.M
	if isPriority {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$[elem].bookedUnitsPriority": -units,
				"timeSlots.$[elem].version":             1,
			},
		}
	} else {
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$[elem].bookedUnitsStandard": -units,
				"timeSlots.$[elem].version":             1,
			},
		}
	}

	res, err := repo.providerColl.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("rollback update error: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("rollback failed: no matching document found or version mismatch")
	}
	return nil
}
