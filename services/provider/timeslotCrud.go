// File: services/provider/timeslot.go
package provider

import (
	"context"
	"fmt"

	"bloomify/models"
)

// GetTimeslots fetches all unblocked timeslots for that provider on the given date.
func (s *DefaultProviderService) GetTimeslots(
	ctx context.Context,
	providerID, date string,
) ([]models.TimeSlot, error) {
	slots, err := s.Timeslot.GetByProviderIDAndDate(ctx, providerID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeslots: %w", err)
	}
	return slots, nil
}

// GetTimeslot retrieves one specific timeslot for a provider on a date.
func (s *DefaultProviderService) GetTimeslot(
	ctx context.Context,
	providerID, slotID, date string,
) (*models.TimeSlot, error) {
	slot, err := s.Timeslot.GetByIDWithDate(ctx, providerID, slotID, date)
	if err != nil {
		return nil, fmt.Errorf("timeslot not found: %w", err)
	}
	return slot, nil
}

// DeleteTimeslot deletes a single timeslot and returns the updated DTO.
func (s *DefaultProviderService) DeleteTimeslot(
	ctx context.Context,
	providerID, slotID, date string,
) (*models.ProviderTimeslotDTO, error) {

	// 1) Verify slot exists and is unbooked
	slot, err := s.Timeslot.GetByIDWithDate(ctx, providerID, slotID, date)
	if err != nil {
		return nil, fmt.Errorf("timeslot not found: %w", err)
	}
	if slot.BookedUnitsStandard > 0 || slot.BookedUnitsPriority > 0 {
		return nil, fmt.Errorf("cannot delete timeslot %s: bookings exist", slotID)
	}

	// 2) Delete it
	if err := s.Timeslot.DeleteByID(ctx, providerID, slotID); err != nil {
		return nil, fmt.Errorf("failed to delete timeslot: %w", err)
	}

	// 3) Remove from provider record
	prov, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	newRefs := prov.TimeSlotRefs[:0]
	for _, ref := range prov.TimeSlotRefs {
		if ref.ID != slotID {
			newRefs = append(newRefs, ref)
		}
	}
	prov.TimeSlotRefs = newRefs

	if err := s.Repo.Update(prov); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// 4) Re-fetch remaining slots for DTO
	remaining, _ := s.Timeslot.GetByProviderIDAndDate(ctx, providerID, date)
	return &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: remaining,
	}, nil
}
