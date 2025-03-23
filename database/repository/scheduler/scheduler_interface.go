package schedulerRepo

import (
	"bloomify/models"
)

// SchedulerRepository defines the interface for data access methods used by the scheduling engine.
type SchedulerRepository interface {
	// GetProviderByID retrieves a provider by its unique ID.
	GetProviderByID(providerID string) (*models.Provider, error)

	// GetBlockedIntervals retrieves blocked intervals for a provider on a given date.
	GetBlockedIntervals(providerID, date string) ([]models.Blocked, error)

	// RemoveBlockedInterval removes a blocked interval record.
	RemoveBlockedInterval(blockedID string) error

	// SumOverlappingBookings aggregates the total booked units for a provider on a given date and time range.
	SumOverlappingBookings(providerID, date string, start, end int) (int, error)

	// SumOverlappingBookingsForStandard aggregates booked units for non-priority bookings.
	SumOverlappingBookingsForStandard(providerID, date string, start, end int) (int, error)

	// SumOverlappingBookingsForPriority aggregates booked units for priority bookings.
	SumOverlappingBookingsForPriority(providerID, date string, start, end int) (int, error)

	// GetAvailableTimeSlots fetches open time slots for a provider on a given date.
	GetAvailableTimeSlots(providerID, date string) ([]models.TimeSlot, error)

	// CreateBooking persists a new booking record.
	CreateBooking(booking *models.Booking) error

	// UpdateBooking modifies an existing booking.
	UpdateBooking(bookingID string, updatedBooking *models.Booking) error

	// CancelBooking removes a booking record from the database.
	CancelBooking(bookingID string) error

	// CreateBlockedInterval persists a new blocked interval record.
	CreateBlockedInterval(blocked *models.Blocked) error

	// UpdateTimeSlotAggregates updates the denormalized aggregates on a provider's timeslot.
	// The parameters include the provider ID, the TimeSlot document (ts), the date, the number of units to update,
	// a flag indicating whether the update is for a priority booking, and the expected version for optimistic locking.
	UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error

	// RollbackTimeSlotAggregates decrements the denormalized aggregates for a provider's timeslot.
	// This is used when a booking is cancelled (for example, due to payment failure).
	RollbackTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error
}
