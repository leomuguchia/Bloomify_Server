package booking

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
)

func (svc *DefaultBookingSessionService) GetAvailableServices() ([]models.Service, error) {
	services := []models.Service{
		{ID: "1", Name: "Babysitting", Icon: "baby-buggy", UnitType: "kids", ProviderTerm: "Babysitters"},
		{ID: "2", Name: "Chauffeuring", Icon: "steering", UnitType: "hour", ProviderTerm: "Chauffeurs"},
		{ID: "3", Name: "Laundry", Icon: "washing-machine", UnitType: "kg", ProviderTerm: "Laundry Service"},
		{ID: "4", Name: "Cleaning", Icon: "broom", UnitType: "hour", ProviderTerm: "Cleaning Professionals"},
		{ID: "5", Name: "Plumbing", Icon: "pipe-wrench", UnitType: "hour", ProviderTerm: "Plumbers"},
		{ID: "6", Name: "Electrical", Icon: "flash", UnitType: "hour", ProviderTerm: "Electricians"},
		{ID: "7", Name: "Delivery", Icon: "truck-delivery", UnitType: "kg", ProviderTerm: "Delivery Personnel"},
		{ID: "8", Name: "Pet Sitting", Icon: "paw", UnitType: "hour", ProviderTerm: "Pet Sitters"},
		{ID: "9", Name: "Tutoring", Icon: "book", UnitType: "hour", ProviderTerm: "Tutors"},
		{ID: "10", Name: "Fitness Training", Icon: "dumbbell", UnitType: "hour", ProviderTerm: "Trainers"},
	}

	return services, nil
}

// CancelSession allows the client to explicitly cancel a booking session.
// It deletes the session data from the cache.
func (s *DefaultBookingSessionService) CancelSession(sessionID string) error {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()
	if err := cacheClient.Del(ctx, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to cancel booking session: %w", err)
	}
	return nil
}
