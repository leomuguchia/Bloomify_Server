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
	customOptionResp *models.CustomOptionResponse, // {Option, Price (per unit) — required}
) (*models.BookingConfirmation, error) {

	// 1) Time bounds
	if booking.Start < slot.Start || booking.End > slot.End {
		return nil, fmt.Errorf(
			"booking time [%d–%d] outside slot [%d–%d]",
			booking.Start, booking.End, slot.Start, slot.End,
		)
	}

	fmt.Printf("CustomOptions in slot: %+v\n", slot.Catalogue.CustomOptions)
	fmt.Printf("User selected option: %q\n", customOptionResp.Option)

	// 2) Base unit price by slot model
	var baseUnitPrice float64
	switch slot.SlotModel {
	case "urgency":
		if slot.Urgency == nil {
			return nil, fmt.Errorf("missing urgency data")
		}
		if booking.Priority {
			baseUnitPrice = slot.BasePrice * (1 + slot.Urgency.PrioritySurchargeRate)
		} else {
			baseUnitPrice = slot.BasePrice
		}

	case "earlybird":
		if slot.EarlyBird == nil {
			return nil, fmt.Errorf("missing earlybird data")
		}
		baseUnitPrice = CalculateEarlyBirdPrice(
			slot.BasePrice,
			*slot.EarlyBird,
			slot.Capacity,
			slot.BookedUnitsStandard,
			booking.Units,
		)

	case "flatrate":
		baseUnitPrice = slot.BasePrice

	default:
		return nil, fmt.Errorf("unknown slot model %q", slot.SlotModel)
	}

	// 3) Lookup the chosen custom option in your trusted catalogue slice
	//    and grab its multiplier.
	if customOptionResp == nil {
		return nil, fmt.Errorf("no custom option provided")
	}
	var multiplier float64
	{
		found := false
		for _, opt := range slot.Catalogue.CustomOptions {
			if opt.Option == customOptionResp.Option {
				multiplier = opt.Multiplier
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid custom option %q", customOptionResp.Option)
		}
	}

	// 4) Recompute per‑unit and validate user’s returned per‑unit price
	expectedPerUnit := math.Round(baseUnitPrice * multiplier)
	if math.Round(customOptionResp.Price*100)/100 != expectedPerUnit {
		return nil, fmt.Errorf(
			"custom option price mismatch: got %.2f, expected %.2f",
			customOptionResp.Price, expectedPerUnit,
		)
	}

	// 5) Total = per‑unit × units
	totalPrice := math.Round(expectedPerUnit*float64(booking.Units)*100) / 100

	// 6) Return confirmation
	return &models.BookingConfirmation{
		BookingID:  uuid.New().String(),
		TotalPrice: totalPrice,
	}, nil
}
