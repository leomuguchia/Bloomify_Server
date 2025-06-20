package booking

import (
	"fmt"
	"math"
	"sort"
	"strings"
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

func getRemainingUnits(ts models.TimeSlot, provider models.Provider) (int, bool) {
	if provider.Profile.ProviderType == "individual" || ts.CapacityMode == models.CapacitySingleUse {
		if len(ts.BookingIDs) > 0 || ts.Blocked {
			return 0, true
		}

		// always enforce to flatrate under individual
		if ts.SlotModel != "flatrate" {
			ts.SlotModel = "flatrate"
		}

		duration := ts.End - ts.Start
		if duration <= 0 {
			return 0, false
		}
		hours := int(math.Round(float64(duration) / 60.0))
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
			return normal - ts.BookedUnitsStandard, true
		case "earlybird", "flatrate":
			return ts.Capacity - ts.BookedUnitsStandard, true
		}
	}

	return 0, false
}

func BuildAvailableSlots(
	enrichedSlots []models.TimeSlot,
	weekStart, weekEnd, now time.Time,
	currency string, units int,
	provider models.Provider,
) ([]models.AvailableSlot, error) {
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

				absEnd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location()).Add(time.Duration(ts.End) * time.Minute)
				if dayStr == now.Format("2006-01-02") && absEnd.Before(now) {
					return
				}

				remaining, ok := getRemainingUnits(ts, provider)
				if !ok || remaining <= 0 {
					return
				}

				slot := models.AvailableSlot{
					ID:                       ts.ID,
					Start:                    ts.Start,
					End:                      ts.End,
					UnitType:                 ts.UnitType,
					Date:                     dayStr,
					Catalogue:                ts.Catalogue,
					OptionPricing:            map[string]float64{},
					CapacityMode:             ts.CapacityMode,
					RegularCapacityRemaining: remaining,
				}

				if slot.Catalogue.Currency == "" {
					slot.Catalogue.Currency = currency
				}

				actualUnits := min(units, remaining)

				switch ts.SlotModel {

				case "urgency":
					if ts.Urgency != nil {
						priorityRemaining := ts.Urgency.ReservedPriority - ts.BookedUnitsPriority
						priorityPrice := ts.BasePrice * (1 + ts.Urgency.PrioritySurchargeRate)

						switch {
						case remaining >= units:
							// All units can be fulfilled with normal stock
							slot.RegularPricePerUnit = math.Round(ts.BasePrice)
							for _, opt := range ts.Catalogue.CustomOptions {
								slot.OptionPricing[opt.Option] = math.Round(ts.BasePrice * opt.Multiplier * float64(units))
							}

						case priorityRemaining >= units:
							// All units available from priority pool
							slot.PriorityPricePerUnit = math.Round(priorityPrice)
							for _, opt := range ts.Catalogue.CustomOptions {
								slot.OptionPricing[opt.Option] = math.Round(priorityPrice * opt.Multiplier * float64(units))
							}
							slot.Message = fmt.Sprintf("You're booking from urgent-use stock — a %.0f%% surcharge applies.", ts.Urgency.PrioritySurchargeRate*100)

						case remaining > 0 && priorityRemaining > 0 && (remaining+priorityRemaining) >= units:
							// Mix of regular and priority
							regularCount := remaining
							priorityCount := units - remaining
							regularTotal := float64(regularCount) * ts.BasePrice
							priorityTotal := float64(priorityCount) * priorityPrice
							slot.RegularPricePerUnit = math.Round((regularTotal + priorityTotal) / float64(units))
							for _, opt := range ts.Catalogue.CustomOptions {
								total := regularTotal*opt.Multiplier + priorityTotal*opt.Multiplier
								slot.OptionPricing[opt.Option] = math.Round(total)
							}
							slot.Message = fmt.Sprintf("Mix of %d normal and %d urgent-use %s — a surcharge applies for the urgent part.", regularCount, priorityCount, slot.UnitType)

						case remaining > 0:
							// Not enough, only partial regular available
							slot.RegularPricePerUnit = math.Round(ts.BasePrice)
							for _, opt := range ts.Catalogue.CustomOptions {
								slot.OptionPricing[opt.Option] = math.Round(ts.BasePrice * opt.Multiplier * float64(remaining))
							}
							slot.Message = fmt.Sprintf("Only %d of %d %s available at normal price — rest are unavailable.", remaining, units, slot.UnitType)

						case priorityRemaining > 0:
							// Not enough, only partial priority available
							slot.PriorityPricePerUnit = math.Round(priorityPrice)
							for _, opt := range ts.Catalogue.CustomOptions {
								slot.OptionPricing[opt.Option] = math.Round(priorityPrice * opt.Multiplier * float64(priorityRemaining))
							}
							slot.Message = fmt.Sprintf("Only %d of %d %s available from urgent-use pool — surcharge applies.", priorityRemaining, units, slot.UnitType)

						default:
							logger.Warn("urgency capacity full", zap.String("slotID", ts.ID))
							return
						}
					}

				case "earlybird":
					if ts.EarlyBird != nil {
						earlyThreshold := int(math.Ceil(float64(ts.Capacity) * 0.25))
						standardThreshold := int(math.Ceil(float64(ts.Capacity) * 0.75))

						earlyCount := 0
						standardCount := 0
						lateCount := 0
						total := 0.0

						for i := 1; i <= actualUnits; i++ {
							unitIndex := ts.BookedUnitsStandard + i
							unitPrice := GetEarlyBirdNextUnitPrice(ts.BasePrice, *ts.EarlyBird, ts.Capacity, ts.BookedUnitsStandard+(i-1))
							total += unitPrice

							if unitIndex <= earlyThreshold {
								earlyCount++
							} else if unitIndex <= standardThreshold {
								standardCount++
							} else {
								lateCount++
							}
						}

						slot.RegularPricePerUnit = math.Round(total / float64(actualUnits))
						for _, opt := range ts.Catalogue.CustomOptions {
							slot.OptionPricing[opt.Option] = math.Round(total * opt.Multiplier)
						}

						var parts []string
						if earlyCount == actualUnits {
							slot.Message = fmt.Sprintf("All %d %s at early-bird discount — save up to %.0f%%", actualUnits, slot.UnitType, ts.EarlyBird.EarlyBirdDiscountRate*100)
						} else {
							if earlyCount > 0 {
								parts = append(parts, fmt.Sprintf("%d at early-bird rate (%.0f%% off)", earlyCount, ts.EarlyBird.EarlyBirdDiscountRate*100))
							}
							if standardCount > 0 {
								parts = append(parts, fmt.Sprintf("%d at standard rate", standardCount))
							}
							if lateCount > 0 {
								parts = append(parts, fmt.Sprintf("%d with late fee (+%.0f%%)", lateCount, ts.EarlyBird.LateSurchargeRate*100))
							}
							slot.Message = fmt.Sprintf("You're booking %d %s: %s", actualUnits, slot.UnitType, strings.Join(parts, ", "))
						}

						if remaining < units {
							slot.Message += fmt.Sprintf(" | Only %d of %d %s available", remaining, units, slot.UnitType)
						}
					}

				case "flatrate":
					price := math.Round(ts.BasePrice)
					slot.RegularPricePerUnit = price
					for _, opt := range ts.Catalogue.CustomOptions {
						slot.OptionPricing[opt.Option] = math.Round(price * opt.Multiplier * float64(actualUnits))
					}
					if remaining < units {
						slot.Message = fmt.Sprintf("Only %d of %d %s available", remaining, units, slot.UnitType)
					}
				}

				if ts.CapacityMode == models.CapacityByUnit &&
					ts.Capacity > 0 &&
					float64(remaining)/float64(ts.Capacity) < 0.3 &&
					slot.Message == "" {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, slot.UnitType)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
