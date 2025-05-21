// File: database/repository/timeslot/interface.go
package timeslotRepo

import (
	"bloomify/database"
	"bloomify/models"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type TimeSlotRepository interface {
	CreateMany(ctx context.Context, slots []models.TimeSlot) ([]string, error)
	DeleteByID(ctx context.Context, providerID, slotID string) error
	GetByProviderIDAndDate(ctx context.Context, providerID, date string) ([]models.TimeSlot, error)
	GetByIDWithDate(ctx context.Context, providerID, slotID, date string) (*models.TimeSlot, error)
	GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error)
	GetMaxAvailableDate(providerID string) (string, error)
	GetTimeSlotByID(providerID, slotID, date string, start, end int) (*models.TimeSlot, error)
	UpdateTimeSlotAggregates(slotID string, date string, units int, priority bool, currentVersion int) error
	SetTimeSlotBlockReason(ctx context.Context, providerID, slotID, date string, blocked bool, blockReason string) error
	RollbackTimeSlotAggregates(slotID string, date string, units int, isPriority bool, minVersion int) error
	TryEmbedBooking(ctx context.Context, providerID, slotID, date, bookingID string, units int, priority bool) error
}

type mongoTimeSlotRepo struct {
	coll *mongo.Collection
}

// NewMongoTimeSlotRepo constructs a new MongoDB TimeSlotRepository.
func NewMongoTimeSlotRepo() TimeSlotRepository {
	db := database.MongoClient.Database("bloomify")
	return &mongoTimeSlotRepo{
		coll: db.Collection("timeslots"),
	}
}
