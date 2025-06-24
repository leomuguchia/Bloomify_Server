package booking

import (
	"bloomify/models"
	"fmt"
	"math"

	"github.com/google/uuid"
)

func ValidateAndBook(
	providerID string,
	slot models.TimeSlot,
	booking models.Booking,
	customOptionResp *models.CustomOptionResponse,
	provider models.Provider,
) (*models.BookingConfirmation, error) {

	// 1. Bounds validation
	if booking.Start < slot.Start || booking.End > slot.End {
		return nil, fmt.Errorf("booking time [%d–%d] outside slot [%d–%d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	// 2. Validate remaining units for freelancer/single-use or capacity logic
	remaining, ok := getRemainingUnits(slot, provider)
	if !ok || remaining <= 0 {
		return nil, fmt.Errorf("slot is no longer available")
	}
	actualUnits := min(booking.Units, remaining)

	// 3. Recompute base unit pricing
	var rawTotal float64
	switch slot.SlotModel {
	case "earlybird":
		if slot.EarlyBird == nil {
			return nil, fmt.Errorf("earlybird pricing model missing data")
		}
		for i := 1; i <= actualUnits; i++ {
			unitPrice := GetEarlyBirdNextUnitPrice(slot.BasePrice, *slot.EarlyBird, slot.Capacity, slot.BookedUnitsStandard+(i-1))
			rawTotal += unitPrice
		}

	case "urgency":
		if slot.Urgency == nil {
			return nil, fmt.Errorf("urgency pricing model missing data")
		}
		if booking.Priority {
			unitPrice := slot.BasePrice * (1 + slot.Urgency.PrioritySurchargeRate)
			rawTotal = unitPrice * float64(actualUnits)
		} else {
			unitPrice := slot.BasePrice
			rawTotal = unitPrice * float64(actualUnits)
		}

	case "flatrate":
		rawTotal = slot.BasePrice * float64(actualUnits)

	default:
		return nil, fmt.Errorf("unknown slot model: %q", slot.SlotModel)
	}

	// 4. Apply multiplier from custom option
	if customOptionResp == nil {
		return nil, fmt.Errorf("custom option missing")
	}
	var multiplier float64
	found := false
	for _, opt := range slot.Catalogue.CustomOptions {
		if opt.Option == customOptionResp.Option {
			multiplier = opt.Multiplier
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("invalid custom option: %q", customOptionResp.Option)
	}

	finalTotal := math.Round(rawTotal * multiplier) // to nearest shilling

	// 5. Validate user-provided price
	expected := finalTotal
	provided := math.Round(customOptionResp.Price)
	if expected != provided {
		var reason string
		switch slot.SlotModel {
		case "earlybird":
			reason = "Pricing has changed due to other users booking before you. Early-bird slots adjust per unit."
		case "urgency":
			reason = "Pricing may have shifted to priority booking due to limited standard capacity."
		case "flatrate":
			reason = "Flat-rate prices do not change. Please ensure you're using the latest app version."
		}
		return nil, fmt.Errorf("custom option price mismatch: got %.2f, expected %.2f. %s", provided, expected, reason)
	}

	// 6. Confirm booking
	return &models.BookingConfirmation{
		BookingID:  uuid.New().String(),
		TotalPrice: expected,
	}, nil
}
