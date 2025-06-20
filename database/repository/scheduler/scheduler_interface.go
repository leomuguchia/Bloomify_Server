package schedulerRepo

import (
	"bloomify/database"
	timeslotRepo "bloomify/database/repository/timeslot"
	"bloomify/models"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type SchedulerRepository interface {
	SumOverlappingBookings(providerID, date string, start, end int, priorityFilter *bool) (int, error)
	CreateBooking(booking *models.Booking) error
	GetBookingByID(ctx context.Context, bookingID string) (*models.Booking, error)
	UpdateBooking(bookingID string, updatedBooking *models.Booking) error
	CancelBooking(bookingID string) error
	BookSingleSlotTransactionally(
		ctx context.Context,
		providerID string,
		date string,
		slot models.TimeSlot,
		booking *models.Booking,
	) error
}

type MongoSchedulerRepo struct {
	providerColl *mongo.Collection
	bookingColl  *mongo.Collection
	timeSlotRepo timeslotRepo.TimeSlotRepository
}

func NewMongoSchedulerRepo(tsRepo timeslotRepo.TimeSlotRepository) SchedulerRepository {
	db := database.MongoClient.Database("bloomify")

	return &MongoSchedulerRepo{
		providerColl: db.Collection("providers"),
		bookingColl:  db.Collection("bookings"),
		timeSlotRepo: tsRepo,
	}
}
