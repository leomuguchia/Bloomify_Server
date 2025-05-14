package schedulerRepo

import (
	"bloomify/models"
	"context"
)

type SchedulerRepository interface {
	SumOverlappingBookings(providerID, date string, start, end int) (int, error)
	SumOverlappingBookingsForStandard(providerID, date string, start, end int) (int, error)
	SumOverlappingBookingsForPriority(providerID, date string, start, end int) (int, error)
	GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error)
	GetMaxAvailableDate(providerID string) (string, error)
	CreateBooking(booking *models.Booking) error
	UpdateBooking(bookingID string, updatedBooking *models.Booking) error
	CancelBooking(bookingID string) error
	UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error
	RollbackEmbeddedTimeSlotAggregates(providerID string, slotID string, date string, units int, isPriority bool, expectedVersion int) error
	EmbedBookingReference(providerID string, slotID string, date string, bookingID string, units int, priority bool) error
	SetEmbeddedTimeSlotBlocked(providerID string, slotID string, date string, blocked bool, reason string) error
	BookSingleSlotTransactionally(
		ctx context.Context,
		providerID string,
		date string,
		slot models.TimeSlot,
		booking *models.Booking,
	) error
}
