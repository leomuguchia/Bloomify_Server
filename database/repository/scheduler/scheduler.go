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
	// SumOverlappingBookings aggregates the total booked units for a provider on a given date and time range.
	SumOverlappingBookings(providerID, date string, start, end int) (int, error)
	// SumOverlappingBookingsForStandard aggregates booked units for non-priority bookings.
	SumOverlappingBookingsForStandard(providerID, date string, start, end int) (int, error)
	// SumOverlappingBookingsForPriority aggregates booked units for priority bookings.
	SumOverlappingBookingsForPriority(providerID, date string, start, end int) (int, error)
	// CreateBooking persists a new booking record.
	CreateBooking(booking *models.Booking) error
	// CreateBlockedInterval persists a new blocked interval record.
	CreateBlockedInterval(blocked *models.Blocked) error
	// UpdateTimeSlotAggregates updates the denormalized aggregates on a provider's timeslot.
	UpdateTimeSlotAggregates(providerID string, ts models.TimeSlot, date string, units int, isPriority bool, expectedVersion int) error
}
