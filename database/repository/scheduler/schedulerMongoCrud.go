package schedulerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

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
