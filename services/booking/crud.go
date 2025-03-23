package booking

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
)

// GetAvailableServices returns a list of 10 available services stored in memory.
func (svc *DefaultBookingSessionService) GetAvailableServices() ([]models.Service, error) {
	services := []models.Service{
		{ID: "1", Name: "Babysitting", Icon: "people"},
		{ID: "2", Name: "Chauffeuring", Icon: "car"},
		{ID: "3", Name: "Laundry", Icon: "water"},
		{ID: "4", Name: "Cleaning", Icon: "broom"},
		{ID: "5", Name: "Plumbing", Icon: "construct"},
		{ID: "6", Name: "Electrical", Icon: "flash"},
		{ID: "7", Name: "Delivery", Icon: "cart"},
		{ID: "8", Name: "Pet Sitting", Icon: "paw"},
		{ID: "9", Name: "Tutoring", Icon: "book"},
		{ID: "10", Name: "Fitness Training", Icon: "fitness"},
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
