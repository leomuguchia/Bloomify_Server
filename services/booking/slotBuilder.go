package booking

import (
	"fmt"
	"math"
	"sort"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func EnrichTimeslots(rawSlots []models.TimeSlot, catalogue models.ServiceCatalogue, logger *zap.Logger) []models.TimeSlot {
	var enriched []models.TimeSlot
	for _, ts := range rawSlots {
		if ts.ID == "" {
			logger.Warn("skipping timeslot with empty ID")
			continue
		}
		enrichedTs, ok := enrichTimeSlot(ts, catalogue, logger)
		if !ok {
			logger.Warn("enrichment failed", zap.String("timeslotID", ts.ID))
			continue
		}
		enriched = append(enriched, enrichedTs)
	}
	return enriched
}

func enrichTimeSlot(ts models.TimeSlot, catalogue models.ServiceCatalogue, logger *zap.Logger) (models.TimeSlot, bool) {
	switch ts.SlotModel {
	case "urgency":
		if ts.Urgency == nil {
			logger.Warn("missing urgency data", zap.String("timeslotID", ts.ID))
			return ts, false
		}
	case "earlybird":
		if ts.EarlyBird == nil {
			logger.Warn("missing earlybird data", zap.String("timeslotID", ts.ID))
			return ts, false
		}
	case "flatrate":
		if ts.Flatrate == nil {
			logger.Warn("missing flatrate data", zap.String("timeslotID", ts.ID))
			return ts, false
		}
	default:
		logger.Warn("unknown slot model", zap.String("timeslotID", ts.ID))
		return ts, false
	}
	// Set the catalogue for pricing and options.
	ts.Catalogue = catalogue
	return ts, true
}

func BuildAvailableSlots(enrichedSlots []models.TimeSlot, catalogue models.ServiceCatalogue, weekStart, weekEnd, now time.Time) ([]models.AvailableSlot, map[string]models.TimeSlot, error) {
	var availableSlots []models.AvailableSlot
	mapping := make(map[string]models.TimeSlot)
	logger := utils.GetLogger()

	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format("2006-01-02")
		for _, ts := range enrichedSlots {
			if ts.Date != dayStr {
				continue
			}
			// Skip timeslots that are blocked.
			if ts.Blocked {
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

				slotID := uuid.New().String()
				slot := models.AvailableSlot{
					ID:            slotID,
					Start:         ts.Start,
					End:           ts.End,
					UnitType:      ts.UnitType,
					Date:          dayStr,
					Catalogue:     ts.Catalogue,
					OptionPricing: make(map[string]float64),
				}
				switch ts.SlotModel {
				case "urgency":
					if ts.Urgency == nil {
						return
					}
					normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority
					remaining := normalCapacity - ts.BookedUnitsStandard
					if remaining <= 0 {
						return
					}
					slot.RegularCapacityRemaining = remaining
					slot.RegularPricePerUnit = ts.Urgency.BasePrice
					for key, modifier := range catalogue.CustomOptions {
						price := ts.Urgency.BasePrice * modifier
						slot.OptionPricing[key] = math.Round(price*100) / 100
					}
					if normalCapacity > 0 && float64(remaining)/float64(normalCapacity) < 0.3 {
						slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
					}
				case "earlybird":
					if ts.EarlyBird == nil {
						return
					}
					remaining := ts.Capacity - ts.BookedUnitsStandard
					if remaining <= 0 {
						return
					}
					nextPrice := GetEarlyBirdNextUnitPrice(*ts.EarlyBird, ts.Capacity, ts.BookedUnitsStandard)
					slot.RegularCapacityRemaining = remaining
					slot.RegularPricePerUnit = nextPrice
					for key, modifier := range catalogue.CustomOptions {
						price := nextPrice * modifier
						slot.OptionPricing[key] = math.Round(price*100) / 100
					}
					if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
						slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
					}
				default: // flatrate or standard.
					if ts.Flatrate == nil {
						return
					}
					remaining := ts.Capacity - ts.BookedUnitsStandard
					if remaining <= 0 {
						return
					}
					slot.RegularCapacityRemaining = remaining
					slot.RegularPricePerUnit = ts.Flatrate.BasePrice
					for key, modifier := range catalogue.CustomOptions {
						price := ts.Flatrate.BasePrice * modifier
						slot.OptionPricing[key] = math.Round(price*100) / 100
					}
					if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
						slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
					}
				}
				availableSlots = append(availableSlots, slot)
				// Map the generated available slot ID to the full timeslot.
				mapping[slotID] = ts
			}(ts)
		}
	}
	sort.Slice(availableSlots, func(i, j int) bool {
		if availableSlots[i].Date == availableSlots[j].Date {
			return availableSlots[i].Start < availableSlots[j].Start
		}
		return availableSlots[i].Date < availableSlots[j].Date
	})
	return availableSlots, mapping, nil
}
