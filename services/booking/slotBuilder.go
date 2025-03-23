package booking

import (
	"fmt"
	"sort"
	"time"

	"bloomify/models"

	"github.com/google/uuid"
)

// buildAvailableSlotsWithMapping constructs AvailableSlot objects and returns a mapping to full TimeSlot objects.
func buildAvailableSlots(enrichedSlots []models.TimeSlot, catalogue models.ServiceCatalogue, weekStart, weekEnd, now time.Time) ([]models.AvailableSlot, map[string]models.TimeSlot, error) {
	var availableSlots []models.AvailableSlot
	mapping := make(map[string]models.TimeSlot)

	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format("2006-01-02")
		for _, ts := range enrichedSlots {
			if ts.Date != dayStr {
				continue
			}
			dayMidnight := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
			absEnd := dayMidnight.Add(time.Duration(ts.End) * time.Minute)
			if dayStr == now.Format("2006-01-02") && absEnd.Before(now) {
				continue
			}

			// Create an AvailableSlot with a unique ID.
			slotID := uuid.New().String()
			slot := models.AvailableSlot{
				ID:              slotID,
				Start:           ts.Start,
				End:             ts.End,
				UnitType:        ts.UnitType,
				Date:            dayStr,
				CustomOptionKey: ts.CustomOptionKey,
				Mode:            ts.Mode,
			}

			switch ts.SlotModel {
			case "urgency":
				if ts.Urgency == nil {
					continue
				}
				normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority
				remaining := normalCapacity - ts.BookedUnitsStandard
				slot.RegularCapacityRemaining = remaining
				slot.RegularPricePerUnit = ts.Urgency.BasePrice
				slot.OptionPricing = make(map[string]float64)
				for key, modifier := range catalogue.CustomOptions {
					slot.OptionPricing[key] = ts.Urgency.BasePrice * modifier
				}
				if normalCapacity > 0 && float64(remaining)/float64(normalCapacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}
			case "earlybird":
				if ts.EarlyBird == nil {
					continue
				}
				usage := ts.BookedUnitsStandard
				remaining := ts.Capacity - usage
				nextPrice := GetEarlyBirdNextUnitPrice(*ts.EarlyBird, ts.Capacity, usage)
				slot.RegularCapacityRemaining = remaining
				slot.RegularPricePerUnit = nextPrice
				slot.OptionPricing = make(map[string]float64)
				for key, modifier := range catalogue.CustomOptions {
					slot.OptionPricing[key] = nextPrice * modifier
				}
				if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}
			default: // flatrate
				if ts.Flatrate == nil {
					continue
				}
				usage := ts.BookedUnitsStandard
				remaining := ts.Capacity - usage
				slot.RegularCapacityRemaining = remaining
				slot.RegularPricePerUnit = ts.Flatrate.BasePrice
				slot.OptionPricing = make(map[string]float64)
				for key, modifier := range catalogue.CustomOptions {
					slot.OptionPricing[key] = ts.Flatrate.BasePrice * modifier
				}
				if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}
			}
			availableSlots = append(availableSlots, slot)
			// Save the mapping from AvailableSlot ID to the full TimeSlot.
			mapping[slotID] = ts
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

// ValidateAndBook validates the booking, applies the custom pricing multiplier,
// calculates the final price, and returns a BookingConfirmation.
func ValidateAndBook(providerID string, slot models.TimeSlot, booking models.Booking, catalogue models.ServiceCatalogue) (*models.BookingConfirmation, error) {
	if booking.Start < slot.Start || booking.End > slot.End {
		return nil, fmt.Errorf("booking time [%d, %d] is not within slot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	modifier := 1.0
	if catalogue.CustomOptions != nil {
		if m, ok := catalogue.CustomOptions[slot.CustomOptionKey]; ok {
			modifier = m
		}
	}

	var basePrice float64
	switch slot.SlotModel {
	case "urgency":
		if slot.Urgency == nil {
			return nil, fmt.Errorf("urgency slot data missing")
		}
		if booking.Priority {
			basePrice = CalculateUrgencyPrice(*slot.Urgency, booking.Units, true)
		} else {
			basePrice = CalculateUrgencyPrice(*slot.Urgency, booking.Units, false)
		}
	case "earlybird":
		basePrice = CalculateEarlyBirdPrice(*slot.EarlyBird, slot.Capacity, slot.BookedUnitsStandard, booking.Units)
	default:
		basePrice = CalculateFlatratePrice(*slot.Flatrate, booking.Units)
	}

	totalPrice := basePrice * modifier
	bookingID := uuid.New().String()
	confirmation := &models.BookingConfirmation{
		BookingID:  bookingID,
		TotalPrice: totalPrice,
		Message:    getCapacityMessage(slot),
	}
	return confirmation, nil
}
