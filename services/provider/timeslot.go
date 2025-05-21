// File: services/provider/timeslot.go
package provider

import (
	"fmt"
	"time"

	"bloomify/models"

	"github.com/gin-gonic/gin"
)

// SetupTimeslots handles a full, week-by-week timeslot setup from the front end.
// It reads models.SetupTimeslotsRequest, expands each WeeklyTemplate into concrete
// TimeSlot instances, persists them via TimeSlotRepo, and links the IDs back to the provider.
func (s *DefaultProviderService) SetupTimeslots(
	c *gin.Context,
	providerID string,
	req models.SetupTimeslotsRequest,
) (*models.ProviderTimeslotDTO, error) {
	ctx := c.Request.Context()

	// 1. Load provider
	prov, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil || prov == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// 2. Expand each week’s template into actual slots
	var allSlots []models.TimeSlot
	for wi, week := range req.Weeks {
		// Parse anchor date
		anchor, err := time.Parse("2006-01-02", week.AnchorDate)
		if err != nil {
			return nil, fmt.Errorf("week %d: invalid anchorDate %q", wi+1, week.AnchorDate)
		}

		// Validate each base slot
		for i, bs := range week.BaseSlots {
			if bs.Date != week.AnchorDate {
				return nil, fmt.Errorf(
					"week %d, slot %d: base slot date %q must equal anchorDate",
					wi+1, i+1, bs.Date,
				)
			}
			if bs.Start >= bs.End {
				return nil, fmt.Errorf(
					"week %d, slot %d: start must be before end",
					wi+1, i+1,
				)
			}
			if prov.Profile.ProviderType == "individual" && bs.Capacity != 1 {
				return nil, fmt.Errorf(
					"week %d, slot %d: individual capacity must be 1; got %d",
					wi+1, i+1, bs.Capacity,
				)
			}
		}

		// Compute week-start (Monday)
		monday := anchor.AddDate(0, 0, -int((anchor.Weekday()+6)%7))

		// Clone base slots onto each active weekday
		for _, wd := range week.ActiveDays {
			delta := (int(wd) - int(monday.Weekday()) + 7) % 7
			slotDate := monday.AddDate(0, 0, delta).Format("2006-01-02")

			for _, base := range week.BaseSlots {
				slot := base                 // copy pricing, capacity, etc.
				slot.ID = ""                 // repo will assign a new UUID
				slot.ProviderID = providerID // link back to provider
				slot.Date = slotDate         // override to actual calendar date
				slot.BookedUnitsStandard = 0 // reset any booking counters
				slot.BookedUnitsPriority = 0
				slot.Blocked = false // ensure open
				slot.BlockReason = ""
				slot.BookingIDs = nil
				allSlots = append(allSlots, slot)
			}
		}
	}

	// 3. Bulk‐insert into your timeslot collection
	ids, err := s.Timeslot.CreateMany(ctx, allSlots)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeslots: %w", err)
	}

	// 4. Activate provider & record timeslot IDs
	prov.Profile.Status = "active"
	prov.TimeSlotIDs = ids
	if err := s.Repo.Update(prov); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// 5. Return the DTO with all created slots
	return &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: allSlots,
	}, nil
}
