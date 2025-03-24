// File: services/provider/timeslot.go
package provider

import (
	"errors"
	"fmt"

	"bloomify/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// SetupTimeslots validates and updates a provider's timeslot data.
// It returns a ProviderTimeslotDTO containing the provider's ID, current status, and the updated timeslots.
func (s *DefaultProviderService) SetupTimeslots(c *gin.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error) {
	// Retrieve the provider record using the repository.
	prov, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil || prov == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Create a set to capture distinct dates.
	dateSet := make(map[string]struct{})
	// Validate each timeslot.
	for _, ts := range req.TimeSlots {
		if ts.Start >= ts.End {
			return nil, errors.New("each timeslot must have a start time less than its end time")
		}
		if ts.Date == "" {
			return nil, errors.New("each timeslot must have a non-empty date")
		}
		// For individual providers, enforce capacity equals 1.
		if prov.Profile.ProviderType == "individual" && ts.Capacity != 1 {
			return nil, fmt.Errorf("individual providers must have a capacity of 1 per timeslot; got %d", ts.Capacity)
		}
		dateSet[ts.Date] = struct{}{}
	}

	// Ensure that the timeslot data spans at least 7 distinct days.
	if len(dateSet) < 7 {
		return nil, errors.New("timeslot setup must cover at least 7 distinct days")
	}

	// Update the provider's timeslots.
	prov.TimeSlots = req.TimeSlots

	// Update the provider status to active (assuming this is stored in the provider's profile).
	prov.Profile.Status = "active"

	// Persist the changes using the repository.
	if err := s.Repo.Update(prov); err != nil {
		return nil, fmt.Errorf("failed to update provider timeslots: %w", err)
	}

	// Build and return a minimal DTO.
	dto := &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: prov.TimeSlots,
	}
	return dto, nil
}

// DeleteTimeslot removes a timeslot from a provider's schedule if no bookings are associated.
// It returns a ProviderTimeslotDTO with the updated timeslot information.
func (s *DefaultProviderService) DeleteTimeslot(c *gin.Context, providerID, timeslotID string) (*models.ProviderTimeslotDTO, error) {
	// Retrieve the provider record.
	prov, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil || prov == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Locate the timeslot index by matching timeslot.ID.
	indexToDelete := -1
	for i, ts := range prov.TimeSlots {
		if ts.ID == timeslotID {
			indexToDelete = i
			break
		}
	}
	if indexToDelete == -1 {
		return nil, fmt.Errorf("timeslot not found")
	}

	// Check if the timeslot is deletable.
	// (For example, if denormalized booking counts indicate existing bookings, reject deletion.)
	if prov.TimeSlots[indexToDelete].BookedUnitsStandard > 0 || prov.TimeSlots[indexToDelete].BookedUnitsPriority > 0 {
		return nil, fmt.Errorf("cannot delete timeslot; bookings exist for this timeslot")
	}

	// Remove the timeslot from the provider's timeslot slice.
	prov.TimeSlots = append(prov.TimeSlots[:indexToDelete], prov.TimeSlots[indexToDelete+1:]...)

	// Persist the update.
	if err := s.Repo.Update(prov); err != nil {
		return nil, fmt.Errorf("failed to update provider timeslots: %w", err)
	}

	// Build and return the minimal DTO.
	dto := &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: prov.TimeSlots,
	}
	return dto, nil
}

// GetTimeslots fetches the timeslots for the given provider.
func (s *DefaultProviderService) GetTimeslots(c *gin.Context, providerID string) ([]models.TimeSlot, error) {
	prov, err := s.Repo.GetByIDWithProjection(providerID, bson.M{"time_slots": 1})
	if err != nil || prov == nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}
	return prov.TimeSlots, nil
}
