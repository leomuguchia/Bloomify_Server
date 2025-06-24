package provider

import (
	"context"
	"fmt"

	"bloomify/models"
)

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

func (s *DefaultProviderService) DeleteTimeslot(
	ctx context.Context,
	providerID, slotID, date string,
) (*models.ProviderTimeslotDTO, error) {
	slot, err := s.Timeslot.GetByIDWithDate(ctx, providerID, slotID, date)
	if err != nil {
		return nil, fmt.Errorf("timeslot not found: %w", err)
	}
	if slot.BookedUnitsStandard > 0 || slot.BookedUnitsPriority > 0 {
		return nil, fmt.Errorf("cannot delete timeslot %s: bookings exist", slotID)
	}

	if err := s.Timeslot.DeleteByID(ctx, providerID, slotID); err != nil {
		return nil, fmt.Errorf("failed to delete timeslot: %w", err)
	}

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

	remaining, _ := s.Timeslot.GetByProviderIDAndDate(ctx, providerID, date)
	return &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: remaining,
	}, nil
}

func (s *DefaultProviderService) VerifyBooking(
	ctx context.Context,
	providerID string,
	date string,
	bookingID string,
) (*models.Booking, error) {
	slots, err := s.Timeslot.GetByProviderIDAndDate(ctx, providerID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeslots: %w", err)
	}

	for _, slot := range slots {
		for _, id := range slot.BookingIDs {
			if id == bookingID {
				booking, err := s.SchedulerRepo.GetBookingByID(ctx, bookingID)
				if err != nil {
					return nil, fmt.Errorf("booking exists in timeslot but not found in DB: %w", err)
				}
				return booking, nil
			}
		}
	}

	return nil, fmt.Errorf("booking not found")
}
