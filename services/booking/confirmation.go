package booking

import (
	"bloomify/models"
	"fmt"

	"github.com/google/uuid"
)

func ValidateAndBook(providerID string, slot models.TimeSlot, booking models.Booking, catalogue models.ServiceCatalogue, customOption *models.CustomOption) (*models.BookingConfirmation, error) {
	if booking.Start < slot.Start || booking.End > slot.End {
		return nil, fmt.Errorf("booking time [%d, %d] is not within slot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	var basePrice float64
	if customOption != nil {
		basePrice = customOption.Price
	} else {
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
	}

	totalPrice := basePrice
	bookingID := uuid.New().String()
	confirmation := &models.BookingConfirmation{
		BookingID:  bookingID,
		TotalPrice: totalPrice,
	}
	return confirmation, nil
}
