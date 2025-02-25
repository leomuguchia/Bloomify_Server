package schedulerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/database"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpdateTimeSlotAggregates updates the denormalized booking counts for a specific timeslot
// on a provider document using optimistic concurrency.
// Parameters:
// - providerID: the ID of the provider.
// - ts: the TimeSlot template we are updating.
// - date: the date of the slot (in "2006-01-02" format).
// - units: number of units to increment (can be negative for cancellation).
// - isPriority: true if the update is for the priority sub-bucket; false for standard.
// - expectedVersion: the current version of the timeslot (must match to update).
func (repo *MongoSchedulerRepo) UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build the filter to match the provider document and the specific timeslot within the time_slots array.
	filter := bson.M{
		"id": providerID,
		"time_slots": bson.M{
			"$elemMatch": bson.M{
				"start":   ts.Start,
				"end":     ts.End,
				"date":    date,
				"version": expectedVersion,
			},
		},
	}

	// Determine which field to increment.
	field := "time_slots.$[elem].booked_units_standard"
	if isPriority {
		field = "time_slots.$[elem].booked_units_priority"
	}

	// Prepare the update: increment the appropriate aggregate and bump the version.
	update := bson.M{
		"$inc": bson.M{
			field:                        units,
			"time_slots.$[elem].version": 1,
		},
	}

	// Create an array filter to target the matching timeslot element.
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.M{
				"elem.start":   ts.Start,
				"elem.end":     ts.End,
				"elem.date":    date,
				"elem.version": expectedVersion,
			},
		},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	// Update the provider document.
	db := database.MongoClient.Database("bloomify")
	providerColl := db.Collection("providers")
	result, err := providerColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("error updating timeslot aggregates: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("timeslot update failed due to version mismatch")
	}
	return nil
}
