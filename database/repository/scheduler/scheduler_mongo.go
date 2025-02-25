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

// SumOverlappingBookings sums the total booked units (regardless of priority) for a provider on a given date and time range.
// A booking is considered overlapping if its "start" is before slotEnd and its "end" is after slotStart.
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

// SumOverlappingBookingsForStandard sums booked units for non-priority bookings.
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

// SumOverlappingBookingsForPriority sums booked units for priority bookings.
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

func (repo *MongoSchedulerRepo) CancelBooking(bookingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Retrieve the booking document.
	var booking models.Booking
	filter := bson.M{"id": bookingID}
	if err := repo.bookingColl.FindOne(ctx, filter).Decode(&booking); err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("booking with id %s not found", bookingID)
		}
		return fmt.Errorf("error fetching booking with id %s: %w", bookingID, err)
	}

	// Step 2: Check if the timeslot has already started.
	// Parse the booking date (assumed to be in "2006-01-02" format).
	bookingDate, err := time.Parse("2006-01-02", booking.Date)
	if err != nil {
		return fmt.Errorf("invalid booking date %q: %w", booking.Date, err)
	}
	// Compute the absolute start time of the booking by adding booking.Start minutes to midnight.
	bookingStartTime := time.Date(bookingDate.Year(), bookingDate.Month(), bookingDate.Day(), 0, 0, 0, 0, time.Local).
		Add(time.Duration(booking.Start) * time.Minute)
	if time.Now().After(bookingStartTime) {
		return fmt.Errorf("cannot cancel booking %s: timeslot has already started", bookingID)
	}

	// Step 3: Delete the booking from the bookings collection.
	delResult, err := repo.bookingColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting booking %s: %w", bookingID, err)
	}
	if delResult.DeletedCount == 0 {
		return fmt.Errorf("booking %s could not be deleted", bookingID)
	}

	// Step 4: Clear the provider schedule.
	// For example, remove any blocked intervals for the same provider and timeslot that
	// were created with reason "capacity reached". This helps free up the slot.
	blockFilter := bson.M{
		"provider_id": booking.ProviderID,
		"date":        booking.Date,
		"start":       booking.Start,
		"end":         booking.End,
		"reason":      "capacity reached",
	}
	_, err = repo.blockedColl.DeleteMany(ctx, blockFilter)
	if err != nil {
		// Log a warning (do not fail the cancellation if cleanup fails).
		fmt.Printf("warning: failed to clear blocked intervals for booking %s: %v\n", bookingID, err)
	}

	return nil
}
