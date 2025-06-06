package booking

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"

	"go.uber.org/zap"
)

func validateServicePlan(plan models.ServicePlan) error {
	if plan.ServiceType == "" {
		return fmt.Errorf("serviceType is required")
	}
	if plan.BookingFor == "" {
		return fmt.Errorf("bookingFor is required")
	}
	if plan.Mode == "" {
		return fmt.Errorf("serviceMode is required")
	}
	if plan.LocationGeo.Type != "Point" {
		return fmt.Errorf("locationGeo.type must be 'Point'")
	}
	// Omit explicit nil check; len(nil) returns 0.
	if len(plan.LocationGeo.Coordinates) < 2 {
		return fmt.Errorf("locationGeo.coordinates must be an array of at least two numbers")
	}
	if plan.Units <= 0 {
		return fmt.Errorf("units must be a positive integer")
	}
	if plan.UnitType == "" {
		return fmt.Errorf("unitType is required")
	}
	return nil
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

func (se *DefaultSchedulingEngine) enrichSingleTimeSlot(slot models.TimeSlot, provider models.Provider) models.TimeSlot {
	// Create a copy to avoid mutation
	enriched := slot
	logger := utils.GetLogger()

	// Merge basic catalogue properties
	enriched.Catalogue.Service.ID = provider.ServiceCatalogue.Service.ID
	enriched.Catalogue.Mode = provider.ServiceCatalogue.Mode
	enriched.Catalogue.Currency = provider.ServiceCatalogue.Currency

	// Create a new slice for merged options
	mergedOptions := make([]models.CustomOption, 0, len(slot.Catalogue.CustomOptions)+len(provider.ServiceCatalogue.CustomOptions))

	// Add existing slot options first
	mergedOptions = append(mergedOptions, slot.Catalogue.CustomOptions...)

	// Merge provider options, overriding existing ones
	for _, providerOpt := range provider.ServiceCatalogue.CustomOptions {
		exists := false
		// Check if option already exists in slot
		for i, slotOpt := range mergedOptions {
			if slotOpt.Option == providerOpt.Option {
				// Overwrite with provider's version
				mergedOptions[i] = providerOpt
				exists = true
				break
			}
		}
		if !exists {
			mergedOptions = append(mergedOptions, providerOpt)
		}
	}

	// Assign the merged options
	enriched.Catalogue.CustomOptions = mergedOptions

	logger.Debug("Enriched timeslot for booking",
		zap.String("slotID", enriched.ID),
		zap.Any("customOptions", enriched.Catalogue.CustomOptions))

	return enriched
}
