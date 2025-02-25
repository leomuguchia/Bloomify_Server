package booking

import (
	"math"

	"bloomify/models"
)

// GetEarlyBirdNextUnitPrice calculates the price for the next unit to be booked in an earlybird slot.
func GetEarlyBirdNextUnitPrice(eb models.EarlyBirdSlotData, capacity int, usage int) float64 {
	nextUnitIndex := usage + 1
	earlyThreshold := int(math.Ceil(float64(capacity) * 0.25))
	standardThreshold := int(math.Ceil(float64(capacity) * 0.75))
	if nextUnitIndex <= earlyThreshold {
		return eb.BasePrice * (1 - eb.EarlyBirdDiscountRate)
	} else if nextUnitIndex <= standardThreshold {
		return eb.BasePrice
	} else {
		return eb.BasePrice * (1 + eb.LateSurchargeRate)
	}
}

// CalculateEarlyBirdPrice computes the total price for booking 'units' in an earlybird slot.
func CalculateEarlyBirdPrice(eb models.EarlyBirdSlotData, capacity, usage, units int) float64 {
	totalPrice := 0.0
	earlyThreshold := int(math.Ceil(float64(capacity) * 0.25))
	standardThreshold := int(math.Ceil(float64(capacity) * 0.75))
	for i := 1; i <= units; i++ {
		unitIndex := usage + i
		if unitIndex <= earlyThreshold {
			totalPrice += eb.BasePrice * (1 - eb.EarlyBirdDiscountRate)
		} else if unitIndex <= standardThreshold {
			totalPrice += eb.BasePrice
		} else {
			totalPrice += eb.BasePrice * (1 + eb.LateSurchargeRate)
		}
	}
	return totalPrice
}

// CalculateUrgencyPrice computes the total price for booking in an urgency slot.
// If isPriority is true, a priority surcharge is applied.
func CalculateUrgencyPrice(urg models.UrgencySlotData, units int, isPriority bool) float64 {
	if isPriority {
		return float64(units) * urg.BasePrice * (1 + urg.PrioritySurchargeRate)
	}
	return float64(units) * urg.BasePrice
}

// CalculateFlatratePrice returns the total price for a flatrate/standard booking.
func CalculateFlatratePrice(fl models.FlatrateSlotData, units int) float64 {
	return float64(units) * fl.BasePrice
}
