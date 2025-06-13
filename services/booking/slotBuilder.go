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

func getRemainingUnits(ts models.TimeSlot) (int, bool) {
	switch ts.CapacityMode {
	case models.CapacitySingleUse:
		if len(ts.BookingIDs) > 0 || ts.Blocked {
			return 0, true
		}
		return 1, true

	case models.CapacityByUnit:
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

	default:
		return 0, false
	}
}

func BuildAvailableSlots(enrichedSlots []models.TimeSlot, weekStart, weekEnd, now time.Time, currency string, units int, provider models.Provider) ([]models.AvailableSlot, error) {
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

				slotID := ts.ID
				slot := models.AvailableSlot{
					ID:            slotID,
					Start:         ts.Start,
					End:           ts.End,
					UnitType:      ts.UnitType,
					Date:          dayStr,
					Catalogue:     ts.Catalogue,
					OptionPricing: make(map[string]float64),
					CapacityMode:  ts.CapacityMode,
				}

				// Determine remaining capacity
				var remaining int
				var ok bool

				if provider.Profile.ProviderType == "individual" {
					// Calculate slot duration in minutes
					durationMinutes := ts.End - ts.Start
					if durationMinutes <= 0 {
						logger.Warn("invalid slot duration", zap.String("slotID", ts.ID), zap.Int("start", ts.Start), zap.Int("end", ts.End))
						return
					}

					// Round duration to nearest hour
					durationHours := int(math.Round(float64(durationMinutes) / 60.0))
					if durationHours <= 0 {
						slot.Message = fmt.Sprintf("Slot duration (%d min) too short for booking", durationMinutes)
						return
					}

					// Set capacity to duration in hours
					remaining = durationHours
					ok = true
					slot.RegularCapacityRemaining = remaining

					slot.Message = fmt.Sprintf(
						"You will book the entire %d %s slot",
						durationHours, ts.UnitType,
					)
				} else {
					remaining, ok = getRemainingUnits(ts)
					if !ok || remaining <= 0 {
						return
					}
					slot.RegularCapacityRemaining = remaining
				}

				if slot.Catalogue.Currency == "" {
					slot.Catalogue.Currency = currency
				}

				switch ts.SlotModel {
				case "urgency":
					if ts.Urgency != nil {
						if remaining >= units {
							slot.RegularPricePerUnit = math.Round(ts.BasePrice)
							for _, option := range ts.Catalogue.CustomOptions {
								price := ts.BasePrice * option.Multiplier
								slot.OptionPricing[option.Option] = math.Round(price)
							}
						} else if slot.PriorityCapacityRemaining >= units {
							slot.PriorityPricePerUnit = math.Round(ts.BasePrice * (1 + ts.Urgency.PrioritySurchargeRate))
							for _, option := range ts.Catalogue.CustomOptions {
								price := slot.PriorityPricePerUnit * option.Multiplier
								slot.OptionPricing[option.Option] = math.Round(price)
							}
						} else if remaining > 0 {
							slot.RegularPricePerUnit = math.Round(ts.BasePrice)
							for _, option := range ts.Catalogue.CustomOptions {
								price := ts.BasePrice * option.Multiplier
								slot.OptionPricing[option.Option] = math.Round(price)
							}
							slot.Message = fmt.Sprintf("Only %d of %d %s available", remaining, units, slot.UnitType)
						} else {
							logger.Warn("capacity is full", zap.String("slotID", ts.ID))
							return
						}
					}
				case "earlybird":
					var price float64
					if ts.EarlyBird != nil {
						price = GetEarlyBirdNextUnitPrice(ts.BasePrice, *ts.EarlyBird, ts.Capacity, ts.BookedUnitsStandard)
					} else {
						price = ts.BasePrice
					}
					price = math.Round(price)
					slot.RegularPricePerUnit = price
					for _, option := range ts.Catalogue.CustomOptions {
						p := price * option.Multiplier
						slot.OptionPricing[option.Option] = math.Round(p)
					}
					if remaining < units {
						slot.Message = fmt.Sprintf("Only %d of %d %s available", remaining, units, slot.UnitType)
					}
				case "flatrate":
					slot.RegularPricePerUnit = math.Round(ts.BasePrice)
					for _, option := range ts.Catalogue.CustomOptions {
						price := ts.BasePrice * option.Multiplier
						slot.OptionPricing[option.Option] = math.Round(price)
					}
					if remaining < units {
						slot.Message = fmt.Sprintf("Only %d of %d %s available", remaining, units, slot.UnitType)
					}
				}

				if ts.CapacityMode == models.CapacityByUnit &&
					ts.Capacity > 0 &&
					float64(remaining)/float64(ts.Capacity) < 0.3 &&
					slot.Message == "" {
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
