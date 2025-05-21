package timeslotRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func (repo *mongoTimeSlotRepo) UpdateTimeSlotAggregates(slotID string, date string, units int, priority bool, currentVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id":      slotID,
		"date":    date,
		"version": currentVersion,
	}

	var update bson.M
	if priority {
		update = bson.M{
			"$inc": bson.M{
				"bookedUnitsPriority": units,
				"version":             1,
			},
		}
	} else {
		update = bson.M{
			"$inc": bson.M{
				"bookedUnitsStandard": units,
				"version":             1,
			},
		}
	}

	res, err := repo.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("timeslot aggregate update failed (version mismatch or not found)")
	}
	return nil
}

func (repo *mongoTimeSlotRepo) SetTimeSlotBlockReason(ctx context.Context, providerID, slotID, date string, blocked bool, blockReason string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id":         slotID,
		"providerId": providerID,
		"date":       date,
	}
	update := bson.M{
		"$set": bson.M{
			"blocked":     blocked,
			"blockReason": blockReason,
		},
	}

	_, err := repo.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to set block state/reason for timeslot: %w", err)
	}
	return nil
}

func (repo *mongoTimeSlotRepo) RollbackTimeSlotAggregates(slotID string, date string, units int, isPriority bool, minVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id":      slotID,
		"date":    date,
		"version": bson.M{"$gt": minVersion},
	}

	var update bson.M
	if isPriority {
		update = bson.M{"$inc": bson.M{"bookedUnitsPriority": -units}}
	} else {
		update = bson.M{"$inc": bson.M{"bookedUnitsStandard": -units}}
	}

	res, err := repo.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to rollback timeslot aggregates: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("rollback update failed; timeslot not found or version condition failed")
	}
	return nil
}

func (r *mongoTimeSlotRepo) TryEmbedBooking(
	ctx context.Context,
	providerID, slotID, date, bookingID string,
	units int,
	priority bool,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	incrementField := "bookedUnitsStandard"
	if priority {
		incrementField = "bookedUnitsPriority"
	}

	// Step 1: Verify the timeslot with that date exists first (pre-check)
	var existingSlot models.TimeSlot
	err := r.coll.FindOne(ctx, bson.M{
		"providerId": providerID,
		"id":         slotID,
		"date":       date,
	}).Decode(&existingSlot)
	if err != nil {
		return fmt.Errorf("timeslot not found for provider %s, slot %s, date %s: %w", providerID, slotID, date, err)
	}

	// Step 2: Proceed to update with full filter including date
	filter := bson.M{
		"providerId": providerID,
		"id":         slotID,
		"date":       date,
	}

	update := bson.M{
		"$addToSet": bson.M{"bookingIds": bookingID},
		"$inc":      bson.M{incrementField: units},
	}

	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to embed booking: %w", err)
	}

	// Step 3: Log matched count to detect filtering issues
	log.Printf("[TryEmbedBooking] UpdateOne matched %d document(s), modified %d", res.MatchedCount, res.ModifiedCount)

	if res.MatchedCount == 0 {
		return fmt.Errorf("no matching slot found for provider %s, slot %s, date %s", providerID, slotID, date)
	}

	return nil
}
