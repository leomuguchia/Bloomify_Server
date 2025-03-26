package booking

import (
	"bloomify/models"
)

// BookingSessionService defines the interface for managing a stateful booking session.
type BookingSessionService interface {
	InitiateSession(plan models.ServicePlan, userID, deviceID, userAgent string) (string, []models.ProviderDTO, error)
	UpdateSession(sessionID string, selectedProviderID string) (*models.BookingSession, error)
	ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlot) (*models.Booking, error)
	CancelSession(sessionID string) error
	GetAvailableServices() ([]models.Service, error)
}

// DefaultBookingSessionService implements BookingSessionService.
type DefaultBookingSessionService struct {
	MatchingSvc     MatchingService
	SchedulerEngine *DefaultSchedulingEngine
}
