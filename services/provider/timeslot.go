package provider

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"bloomify/models"
)

var dayOrder = map[string]int{
	"Mon": 0, "Tue": 1, "Wed": 2, "Thu": 3, "Fri": 4, "Sat": 5, "Sun": 6,
}

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

	var allSlots []models.TimeSlot

	for wi, week := range req.Weeks {
		// Parse anchor date
		weekStart, err := time.Parse("2006-01-02", week.StartDate)
		if err != nil {
			return nil, fmt.Errorf("week %d: invalid startDate %q", wi+1, week.StartDate)
		}

		for i, bs := range week.BaseSlots {
			// Reset stateful fields
			bs.BookedUnitsStandard = 0
			bs.BookedUnitsPriority = 0
			bs.Blocked = false
			bs.BlockReason = ""
			bs.BookingIDs = nil

			// Infer capacity mode
			if bs.CapacityMode == "" {
				if prov.Profile.ProviderType == "freelancer" {
					bs.CapacityMode = models.CapacitySingleUse
				} else {
					bs.CapacityMode = models.CapacityByUnit
				}
			}

			// Validate slot
			if err := validateSlotStructure(bs, *prov, wi, i); err != nil {
				return nil, err
			}

			// Expand across active days
			for _, wd := range week.ActiveDays {
				dayIdx, ok := dayOrder[strings.Title(wd)]
				if !ok {
					return nil, fmt.Errorf("week %d: invalid weekday %q", wi+1, wd)
				}

				slotDate := weekStart.AddDate(0, 0, dayIdx).Format("2006-01-02")

				slot := bs
				slot.ID = ""
				slot.ProviderID = providerID
				slot.Date = slotDate
				allSlots = append(allSlots, slot)
			}
		}
	}

	// 2. Persist
	ids, err := s.Timeslot.CreateMany(ctx, allSlots)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeslots: %w", err)
	}

	// 3. Update provider
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

	return &models.ProviderTimeslotDTO{
		ID:        prov.ID,
		Status:    prov.Profile.Status,
		TimeSlots: allSlots,
	}, nil
}

func validateSlotStructure(slot models.TimeSlot, provider models.Provider, weekIdx, slotIdx int) error {
	if slot.Start >= slot.End {
		return fmt.Errorf("week %d, slot %d: start must be before end", weekIdx+1, slotIdx+1)
	}
	if slot.CapacityMode == models.CapacityByUnit && slot.Capacity < 1 {
		return fmt.Errorf("week %d, slot %d: CapacityByUnit requires capacity >= 1", weekIdx+1, slotIdx+1)
	}

	if _, ok := getRemainingUnits(slot, provider); !ok {
		return fmt.Errorf("week %d, slot %d: invalid slot configuration", weekIdx+1, slotIdx+1)
	}
	return nil
}

func getRemainingUnits(ts models.TimeSlot, provider models.Provider) (int, bool) {
	if provider.Profile.ProviderType == "freelancer" || ts.CapacityMode == models.CapacitySingleUse {
		if ts.SlotModel != "flatrate" {
			ts.SlotModel = "flatrate"
		}
		duration := ts.End - ts.Start
		if duration <= 0 {
			return 0, false
		}
		hours := int(math.Floor(float64(duration) / 60.0))
		if hours <= 0 {
			return 0, false
		}
		return hours, true
	}

	if ts.CapacityMode == models.CapacityByUnit {
		switch ts.SlotModel {
		case "urgency":
			if ts.Urgency == nil || !ts.Urgency.PriorityActive {
				return 0, false
			}
			normal := ts.Capacity - ts.Urgency.ReservedPriority
			return normal, true
		case "earlybird", "flatrate":
			return ts.Capacity, true
		}
	}
	return 0, false
}
