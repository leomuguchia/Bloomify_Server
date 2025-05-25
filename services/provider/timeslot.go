// File: services/provider/timeslot.go
package provider

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
)

// SetupTimeslots handles a full, week-by-week timeslot setup from the front end.
// It reads models.SetupTimeslotsRequest, expands each WeeklyTemplate into concrete
// TimeSlot instances, persists them via TimeSlotRepo, and links the IDs back to the provider.
func (s *DefaultProviderService) SetupTimeslots(
	ctx context.Context,
	providerID string,
	req models.SetupTimeslotsRequest,
) (*models.ProviderTimeslotDTO, error) {
	// 1. Load provider
	prov, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil || prov == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// 2. Expand each week’s template into actual slots
	var allSlots []models.TimeSlot
	for wi, week := range req.Weeks {
		anchor, err := time.Parse("2006-01-02", week.AnchorDate)
		if err != nil {
			return nil, fmt.Errorf("week %d: invalid anchorDate %q", wi+1, week.AnchorDate)
		}

		for i, bs := range week.BaseSlots {
			if bs.Date != week.AnchorDate {
				return nil, fmt.Errorf("week %d, slot %d: base slot date %q must equal anchorDate", wi+1, i+1, bs.Date)
			}
			if bs.Start >= bs.End {
				return nil, fmt.Errorf("week %d, slot %d: start must be before end", wi+1, i+1)
			}
			if prov.Profile.ProviderType == "individual" && bs.Capacity != 1 {
				return nil, fmt.Errorf("week %d, slot %d: individual capacity must be 1; got %d", wi+1, i+1, bs.Capacity)
			}
		}

		monday := anchor.AddDate(0, 0, -int((anchor.Weekday()+6)%7))

		for _, wd := range week.ActiveDays {
			delta := (int(wd) - int(monday.Weekday()) + 7) % 7
			slotDate := monday.AddDate(0, 0, delta).Format("2006-01-02")

			for _, base := range week.BaseSlots {
				slot := base
				slot.ID = ""
				slot.ProviderID = providerID
				slot.Date = slotDate
				slot.BookedUnitsStandard = 0
				slot.BookedUnitsPriority = 0
				slot.Blocked = false
				slot.BlockReason = ""
				slot.BookingIDs = nil
				allSlots = append(allSlots, slot)
			}
		}
	}

	// 3. Insert into DB
	ids, err := s.Timeslot.CreateMany(ctx, allSlots)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeslots: %w", err)
	}

	// 4. Activate provider & add MinimalSlotDTOs
	prov.Profile.Status = "active"
	var slotRefs []models.MinimalSlotDTO
	for i, id := range ids {
		slotRefs = append(slotRefs, models.MinimalSlotDTO{
			ID:        id,
			Date:      allSlots[i].Date,
			Start:     allSlots[i].Start,
			End:       allSlots[i].End,
			SlotModel: allSlots[i].SlotModel,
		})
	}
	prov.TimeSlotRefs = slotRefs

	if err := s.Repo.Update(prov); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// 5. Return DTO
	return &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: allSlots,
	}, nil
}
