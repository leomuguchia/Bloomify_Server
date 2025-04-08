package schedulerRepo

import (
	"bloomify/database"
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// MongoSchedulerRepo implements SchedulerRepository using MongoDB.
type MongoSchedulerRepo struct {
	providerColl *mongo.Collection
	bookingColl  *mongo.Collection
}

// NewMongoSchedulerRepo constructs a new instance of MongoSchedulerRepo.
func NewMongoSchedulerRepo() SchedulerRepository {
	db := database.MongoClient.Database("bloomify")
	return &MongoSchedulerRepo{
		providerColl: db.Collection("providers"),
		bookingColl:  db.Collection("bookings"),
	}
}

// CreateBooking inserts a new booking document.
func (repo *MongoSchedulerRepo) CreateBooking(booking *models.Booking) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := repo.bookingColl.InsertOne(ctx, booking)
	if err != nil {
		return fmt.Errorf("error creating booking: %w", err)
	}
	return nil
}

// UpdateBooking modifies an existing booking document.
func (repo *MongoSchedulerRepo) UpdateBooking(bookingID string, updatedBooking *models.Booking) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": bookingID}
	update := bson.M{"$set": updatedBooking}
	_, err := repo.bookingColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating booking %s: %w", bookingID, err)
	}
	return nil
}

// CancelBooking removes a booking record from the database.
func (repo *MongoSchedulerRepo) CancelBooking(bookingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": bookingID}
	_, err := repo.bookingColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting booking %s: %w", bookingID, err)
	}
	return nil
}

func (repo *MongoSchedulerRepo) EmbedBookingReference(providerID, slotID, date, bookingID string, units int, priority bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build the filter that locates the provider document with the desired timeslot.
	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      slotID,
				"date":    date,
				"blocked": false,
			},
		},
	}
	// Log the filter for debugging.
	utils.GetLogger().Debug("EmbedBookingReference filter",
		zap.Any("filter", filter))

	// Build the update to push the booking ID and increment the aggregate.
	update := bson.M{
		"$push": bson.M{"timeSlots.$.bookingIds": bookingID},
	}
	if priority {
		update["$inc"] = bson.M{"timeSlots.$.bookedUnitsPriority": units}
	} else {
		update["$inc"] = bson.M{"timeSlots.$.bookedUnitsStandard": units}
	}
	// Log the update document.
	utils.GetLogger().Debug("EmbedBookingReference update",
		zap.Any("update", update))

	res, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		utils.GetLogger().Error("EmbedBookingReference failed during update", zap.Error(err))
		return fmt.Errorf("failed to embed booking reference: %w", err)
	}
	// Log the result of the update.
	utils.GetLogger().Debug("EmbedBookingReference update result",
		zap.Int64("matchedCount", res.MatchedCount),
		zap.Int64("modifiedCount", res.ModifiedCount))
	if res.MatchedCount == 0 {
		// Log error details if nothing matched.
		utils.GetLogger().Error("EmbedBookingReference found no matching timeslot",
			zap.String("providerID", providerID),
			zap.String("slotID", slotID),
			zap.String("date", date))
		return fmt.Errorf("no matching timeslot found or slot is blocked")
	}
	return nil
}

func (repo *MongoSchedulerRepo) SetEmbeddedTimeSlotBlocked(providerID, slotID, date string, blocked bool, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":   slotID,
				"date": date,
			},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"timeSlots.$.blocked":     blocked,
			"timeSlots.$.blockReason": reason,
		},
	}
	_, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update blocked flag for timeslot: %w", err)
	}
	return nil
}
