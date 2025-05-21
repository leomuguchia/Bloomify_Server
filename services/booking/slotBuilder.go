package booking

import (
	"fmt"
	"math"
	"sort"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.uber.org/zap"
)

func EnrichTimeslots(rawSlots []models.TimeSlot, catalogue models.ServiceCatalogue, logger *zap.Logger) []models.TimeSlot {
	enriched := make([]models.TimeSlot, len(rawSlots))

	for i := range rawSlots {
		ts := &rawSlots[i]

		if ts.ID == "" {
			logger.Warn("skipping empty ID slot", zap.Int("index", i))
			continue
		}

		// Merge provider catalogue into timeslot
		ts.Catalogue.Service.ID = catalogue.Service.ID
		ts.Catalogue.Mode = catalogue.Mode
		ts.Catalogue.CustomOptions = append(
			ts.Catalogue.CustomOptions, // Preserve existing
			catalogue.CustomOptions..., // Add provider options
		)
		ts.Catalogue.Currency = catalogue.Currency

		enriched[i] = *ts
	}
	return enriched
}

// Abstracted logic for computing remaining units.
func getRemainingUnits(ts models.TimeSlot) (int, bool) {
	switch ts.SlotModel {
	case "urgency":
		if ts.Urgency == nil {
			return 0, false
		}
		normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority
		return normalCapacity - ts.BookedUnitsStandard, true
	case "earlybird", "flatrate":
		return ts.Capacity - ts.BookedUnitsStandard, true
	default:
		return 0, false
	}
}

func BuildAvailableSlots(enrichedSlots []models.TimeSlot, weekStart, weekEnd, now time.Time, currency string) ([]models.AvailableSlot, error) {
	var availableSlots []models.AvailableSlot
	logger := utils.GetLogger()

	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format("2006-01-02")
		for _, ts := range enrichedSlots {
			if ts.Date != dayStr || ts.Blocked {
				continue
			}

			func(ts models.TimeSlot) {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic processing timeslot", zap.Any("recover", r), zap.Any("timeslot", ts))
					}
				}()

				dayMidnight := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
				absEnd := dayMidnight.Add(time.Duration(ts.End) * time.Minute)
				if dayStr == now.Format("2006-01-02") && absEnd.Before(now) {
					return
				}

				remaining, ok := getRemainingUnits(ts)
				if !ok || remaining <= 0 {
					return
				}

				slotID := ts.ID
				slot := models.AvailableSlot{
					ID:            slotID,
					Start:         ts.Start,
					End:           ts.End,
					UnitType:      ts.UnitType,
					Date:          dayStr,
					Catalogue:     ts.Catalogue,
					OptionPricing: make(map[string]float64),
				}

				slot.RegularCapacityRemaining = remaining
				if slot.Catalogue.Currency == "" {
					slot.Catalogue.Currency = currency
				}

				switch ts.SlotModel {
				case "urgency":
					slot.PriorityPricePerUnit = ts.BasePrice * (1 + ts.Urgency.PrioritySurchargeRate)
					for _, option := range ts.Catalogue.CustomOptions {
						price := slot.PriorityPricePerUnit * option.Multiplier
						slot.OptionPricing[option.Option] = math.Round(price*100) / 100
					}
				case "earlybird":
					nextPrice := GetEarlyBirdNextUnitPrice(ts.BasePrice, *ts.EarlyBird, ts.Capacity, ts.BookedUnitsStandard)
					slot.RegularPricePerUnit = nextPrice
					for _, option := range ts.Catalogue.CustomOptions {
						price := nextPrice * option.Multiplier
						slot.OptionPricing[option.Option] = math.Round(price*100) / 100
					}
				case "flatrate":
					slot.RegularPricePerUnit = ts.BasePrice
					for _, option := range ts.Catalogue.CustomOptions {
						price := ts.BasePrice * option.Multiplier
						slot.OptionPricing[option.Option] = math.Round(price*100) / 100
					}
				}

				if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}

				availableSlots = append(availableSlots, slot)
			}(ts)
		}
	}

	sort.Slice(availableSlots, func(i, j int) bool {
		if availableSlots[i].Date == availableSlots[j].Date {
			return availableSlots[i].Start < availableSlots[j].Start
		}
		return availableSlots[i].Date < availableSlots[j].Date
	})

	return availableSlots, nil
}
