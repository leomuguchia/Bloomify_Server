package schedulerRepo

import (
	"bloomify/database"
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoSchedulerRepo implements SchedulerRepository using MongoDB.
type MongoSchedulerRepo struct {
	providerColl *mongo.Collection
	blockedColl  *mongo.Collection
	bookingColl  *mongo.Collection
}

// NewMongoSchedulerRepo constructs a new instance of MongoSchedulerRepo.
func NewMongoSchedulerRepo() SchedulerRepository {
	db := database.MongoClient.Database("bloomify")
	return &MongoSchedulerRepo{
		providerColl: db.Collection("providers"),
		blockedColl:  db.Collection("blocked"),
		bookingColl:  db.Collection("bookings"),
	}
}

// GetProviderByID retrieves a provider document by ID.
func (repo *MongoSchedulerRepo) GetProviderByID(providerID string) (*models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var provider models.Provider
	filter := bson.M{"id": providerID}
	if err := repo.providerColl.FindOne(ctx, filter).Decode(&provider); err != nil {
		return nil, fmt.Errorf("error fetching provider with id %s: %w", providerID, err)
	}
	return &provider, nil
}

// GetBlockedIntervals retrieves all blocked intervals for a given provider and date.
func (repo *MongoSchedulerRepo) GetBlockedIntervals(providerID, date string) ([]models.Blocked, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"provider_id": providerID, "date": date}
	cursor, err := repo.blockedColl.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error fetching blocked intervals: %w", err)
	}
	defer cursor.Close(ctx)

	var blocked []models.Blocked
	for cursor.Next(ctx) {
		var b models.Blocked
		if err := cursor.Decode(&b); err != nil {
			return nil, fmt.Errorf("error decoding blocked interval: %w", err)
		}
		blocked = append(blocked, b)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return blocked, nil
}

// GetAvailableTimeSlots fetches available timeslots for a provider for a given date.
// We assume each provider document contains a "timeSlots" array field.
func (repo *MongoSchedulerRepo) GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var provider models.Provider
	filter := bson.M{"id": providerID}
	if err := repo.providerColl.FindOne(ctx, filter).Decode(&provider); err != nil {
		return nil, fmt.Errorf("error fetching provider with id %s: %w", providerID, err)
	}

	// Filter the provider's timeslots by the given date.
	var available []models.TimeSlot
	for _, ts := range provider.TimeSlots {
		if ts.Date == date {
			available = append(available, ts)
		}
	}
	return available, nil
}

// SumOverlappingBookings aggregates the total booked units (regardless of priority)
// for a provider on a given date and time range.
func (repo *MongoSchedulerRepo) SumOverlappingBookings(providerID, date string, slotStart, slotEnd int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"provider_id": providerID,
		"date":        date,
		"start":       bson.M{"$lt": slotEnd},
	}
	cursor, err := repo.bookingColl.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error finding overlapping bookings: %w", err)
	}
	defer cursor.Close(ctx)

	totalUnits := 0
	for cursor.Next(ctx) {
		var booking models.Booking
		if err := cursor.Decode(&booking); err != nil {
			return 0, fmt.Errorf("error decoding booking: %w", err)
		}
		if booking.End > slotStart {
			totalUnits += booking.Units
		}
	}
	return totalUnits, nil
}

// SumOverlappingBookingsForStandard aggregates booked units for non-priority bookings.
func (repo *MongoSchedulerRepo) SumOverlappingBookingsForStandard(providerID, date string, slotStart, slotEnd int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"provider_id": providerID,
		"date":        date,
		"start":       bson.M{"$lt": slotEnd},
		"priority":    false,
	}
	cursor, err := repo.bookingColl.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error finding overlapping standard bookings: %w", err)
	}
	defer cursor.Close(ctx)

	totalUnits := 0
	for cursor.Next(ctx) {
		var booking models.Booking
		if err := cursor.Decode(&booking); err != nil {
			return 0, fmt.Errorf("error decoding standard booking: %w", err)
		}
		if booking.End > slotStart {
			totalUnits += booking.Units
		}
	}
	return totalUnits, nil
}

// SumOverlappingBookingsForPriority aggregates booked units for priority bookings.
func (repo *MongoSchedulerRepo) SumOverlappingBookingsForPriority(providerID, date string, slotStart, slotEnd int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"provider_id": providerID,
		"date":        date,
		"start":       bson.M{"$lt": slotEnd},
		"priority":    true,
	}
	cursor, err := repo.bookingColl.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error finding overlapping priority bookings: %w", err)
	}
	defer cursor.Close(ctx)

	totalUnits := 0
	for cursor.Next(ctx) {
		var booking models.Booking
		if err := cursor.Decode(&booking); err != nil {
			return 0, fmt.Errorf("error decoding priority booking: %w", err)
		}
		if booking.End > slotStart {
			totalUnits += booking.Units
		}
	}
	return totalUnits, nil
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

	var booking models.Booking
	filter := bson.M{"id": bookingID}
	if err := repo.bookingColl.FindOne(ctx, filter).Decode(&booking); err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("booking with id %s not found", bookingID)
		}
		return fmt.Errorf("error fetching booking with id %s: %w", bookingID, err)
	}

	bookingDate, err := time.Parse("2006-01-02", booking.Date)
	if err != nil {
		return fmt.Errorf("invalid booking date %q: %w", booking.Date, err)
	}
	bookingStartTime := time.Date(bookingDate.Year(), bookingDate.Month(), bookingDate.Day(), 0, 0, 0, 0, time.Local).
		Add(time.Duration(booking.Start) * time.Minute)
	if time.Now().After(bookingStartTime) {
		return fmt.Errorf("cannot cancel booking %s: timeslot has already started", bookingID)
	}

	delResult, err := repo.bookingColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting booking %s: %w", bookingID, err)
	}
	if delResult.DeletedCount == 0 {
		return fmt.Errorf("booking %s could not be deleted", bookingID)
	}

	blockFilter := bson.M{
		"provider_id": booking.ProviderID,
		"date":        booking.Date,
		"start":       booking.Start,
		"end":         booking.End,
		"reason":      "capacity reached",
	}
	_, err = repo.blockedColl.DeleteMany(ctx, blockFilter)
	if err != nil {
		fmt.Printf("warning: failed to clear blocked intervals for booking %s: %v\n", bookingID, err)
	}

	return nil
}

// CreateBlockedInterval inserts a new blocked interval document.
func (repo *MongoSchedulerRepo) CreateBlockedInterval(blocked *models.Blocked) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := repo.blockedColl.InsertOne(ctx, blocked)
	if err != nil {
		return fmt.Errorf("error creating blocked interval: %w", err)
	}
	return nil
}

// RemoveBlockedInterval removes a blocked interval record.
func (repo *MongoSchedulerRepo) RemoveBlockedInterval(blockedID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": blockedID}
	_, err := repo.blockedColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error removing blocked interval with id %s: %w", blockedID, err)
	}
	return nil
}
