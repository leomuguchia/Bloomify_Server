package booking

import (
	"bloomify/models"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// SubscriptionBookingResult aggregates successful bookings and errors for a subscription.
type SubscriptionBookingResult struct {
	SuccessfulBookings []models.Booking
	Errors             []error
}

func (se *DefaultSchedulingEngine) BookSlot(provider models.Provider, req models.BookingRequest) (*models.Booking, error) {
	log.Printf("[BookSlot] Starting booking process for user %s with provider %s and booking request %+v", req.UserID, provider.ID, req)
	// Subscription booking branch.
	if req.Subscription {
		log.Printf("[BookSlot] Detected subscription booking")
		baseBooking := models.Booking{
			ID:           uuid.New().String(),
			ProviderID:   provider.ID,
			UserID:       req.UserID,
			Units:        req.Units,
			Start:        req.Start,
			End:          req.End,
			UnitType:     req.UnitType,
			Priority:     req.Priority,
			CustomOption: req.CustomOption,
			UserPayment:  req.UserPayment,
		}
		log.Printf("[BookSlot] Created base subscription booking: %+v", baseBooking)
		return se.bookSubscriptionSlots(provider, baseBooking, req.SubscriptionDetails)
	}

	log.Printf("[BookSlot] Detected one-off booking")

	if req.Date == "" || req.Start == 0 || req.End == 0 {
		log.Printf("[BookSlot] Invalid request: missing date/start/end. Date: %s, Start: %d, End: %d", req.Date, req.Start, req.End)
		return nil, fmt.Errorf("missing date or time details for one-off booking")
	}

	log.Printf("[BookSlot] Fetching available timeslots for date: %s", req.Date)
	daySlots, err := se.Repo.GetAvailableTimeSlots(provider.ID, req.Date)
	if err != nil {
		log.Printf("[BookSlot] Error fetching timeslots: %v", err)
		return nil, fmt.Errorf("failed to fetch timeslots for date %s: %w", req.Date, err)
	}
	log.Printf("[BookSlot] Retrieved %d timeslots", len(daySlots))

	if len(daySlots) == 0 {
		log.Printf("[BookSlot] No available timeslots found for date %s", req.Date)
		return nil, fmt.Errorf("no available timeslots for date %s", req.Date)
	}

	var selectedSlot *models.TimeSlot
	for _, ts := range daySlots {
		if ts.Start == req.Start && ts.End == req.End {
			selectedSlot = &ts
			break
		}
	}
	if selectedSlot == nil {
		log.Printf("[BookSlot] No matching timeslot found for [%d, %d] on %s", req.Start, req.End, req.Date)
		return nil, fmt.Errorf("no matching timeslot available for requested time [%d, %d] on %s", req.Start, req.End, req.Date)
	}

	log.Printf("[BookSlot] Selected slot: %+v", *selectedSlot)

	if selectedSlot.UnitType != req.UnitType {
		log.Printf("[BookSlot] Unit type mismatch: requested %s, found %s", req.UnitType, selectedSlot.UnitType)
		return nil, fmt.Errorf("unit type mismatch: requested %s, available %s", req.UnitType, selectedSlot.UnitType)
	}

	booking := &models.Booking{
		ID:           uuid.New().String(),
		ProviderID:   provider.ID,
		UserID:       req.UserID,
		Date:         req.Date,
		Start:        req.Start,
		End:          req.End,
		Units:        req.Units,
		UnitType:     req.UnitType,
		Priority:     req.Priority,
		CreatedAt:    time.Now(),
		CustomOption: req.CustomOption,
		UserPayment:  req.UserPayment,
	}

	log.Printf("[BookSlot] Creating booking: %+v", booking)

	err = se.bookSingleSlot(provider, req.Date, *selectedSlot, booking, req.CustomOption)
	if err != nil {
		log.Printf("[BookSlot] Error booking slot: %v", err)
		return nil, err
	}

	log.Printf("[BookSlot] Booking successful. Booking ID: %s", booking.ID)
	return booking, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
