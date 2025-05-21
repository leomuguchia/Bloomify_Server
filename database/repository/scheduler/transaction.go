package schedulerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

func (repo *MongoSchedulerRepo) EmbedBookingReference(
	providerID, slotID, date, bookingID string,
	units int, priority bool,
) error {
	// DEPRECATED: this should be handled by the timeSlotRepo interface.
	return fmt.Errorf("EmbedBookingReference is deprecated; use TryEmbedBooking via timeSlotRepo instead")
}

func (repo *MongoSchedulerRepo) BookSingleSlotTransactionally(
	ctx context.Context,
	providerID string,
	date string,
	slot models.TimeSlot,
	booking *models.Booking,
) error {
	// Start MongoDB session from booking collection (not timeslot internal client)
	client := repo.bookingColl.Database().Client()
	sess, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("could not start mongo session: %w", err)
	}
	defer sess.EndSession(ctx)

	txnFn := func(sc mongo.SessionContext) error {
		// Insert booking document
		if _, err := repo.bookingColl.InsertOne(sc, booking); err != nil {
			return fmt.Errorf("insert booking failed: %w", err)
		}

		// Embed booking into timeslot using its parts
		if err := repo.timeSlotRepo.TryEmbedBooking(sc, providerID, slot.ID, date, booking.ID, booking.Units, booking.Priority); err != nil {
			return fmt.Errorf("failed to embed booking into time slot: %w", err)
		}

		return nil
	}

	if err := mongo.WithSession(ctx, sess, func(sc mongo.SessionContext) error {
		if err := sc.StartTransaction(); err != nil {
			return err
		}
		if err := txnFn(sc); err != nil {
			_ = sc.AbortTransaction(sc)
			return err
		}
		return sc.CommitTransaction(sc)
	}); err != nil {
		return fmt.Errorf("booking transaction failed: %w", err)
	}

	return nil
}

func (repo *MongoSchedulerRepo) SetTimeSlotBlocked(
	providerID, slotID, date string,
	blocked bool, reason string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := repo.timeSlotRepo.SetTimeSlotBlockReason(ctx, providerID, slotID, date, blocked, reason)
	if err != nil {
		return fmt.Errorf("failed to update blocked flag for time slot: %w", err)
	}
	return nil
}
