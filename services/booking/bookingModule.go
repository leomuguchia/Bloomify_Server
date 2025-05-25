package booking

import (
	"bloomify/models"
	"fmt"
	"log"

	"slices"

	"github.com/google/uuid"
)

// SubscriptionBookingResult aggregates successful bookings and errors for a subscription.
type SubscriptionBookingResult struct {
	SuccessfulBookings []models.Booking
	Errors             []error
}

func (se *DefaultSchedulingEngine) BookSlot(provider models.Provider, req models.BookingRequest) (*models.PublicBookingData, error) {
	log.Printf("[BookSlot] Starting booking process for user %s with provider %s", req.UserID, provider.ID)

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
			Mode:         req.Mode,
		}
		return se.bookSubscriptionSlots(provider, baseBooking, req.SubscriptionDetails)
	}

	if req.SlotID == "" {
		return nil, fmt.Errorf("missing slot ID for one-off booking")
	}

	log.Printf("[BookSlot] Fetching timeslot by ID: %s (date: %s, start: %d, end: %d)", req.SlotID, req.Date, req.Start, req.End)
	selectedSlot, err := se.TimeslotsRepo.GetTimeSlotByID(provider.ID, req.SlotID, req.Date, req.Start, req.End)
	if err != nil {
		log.Printf("[BookSlot] Error fetching timeslot by ID: %v", err)
		return nil, err
	}
	log.Printf("[BookSlot] Found timeslot: %+v", *selectedSlot)

	// Enrich with latest provider data
	enrichedSlot := se.enrichSingleTimeSlot(*selectedSlot, provider)
	log.Printf("[BookSlot] Enriched slot options: %+v", enrichedSlot.Catalogue.CustomOptions)

	valid := false
	for _, opt := range enrichedSlot.Catalogue.CustomOptions {
		if opt.Option == req.CustomOption.Option {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid custom option %q", req.CustomOption.Option)
	}

	booking := &models.Booking{
		ID:           uuid.New().String(),
		ProviderID:   provider.ID,
		UserID:       req.UserID,
		Date:         enrichedSlot.Date,
		Start:        enrichedSlot.Start,
		End:          enrichedSlot.End,
		Units:        req.Units,
		UnitType:     enrichedSlot.UnitType,
		Priority:     req.Priority,
		CustomOption: req.CustomOption,
		UserPayment:  req.UserPayment,
		ServiceType:  enrichedSlot.Catalogue.Service.ID,
		Mode:         req.Mode,
	}

	log.Printf("[BookSlot] Creating booking record: %+v", booking)

	if err := se.bookSingleSlot(provider, selectedSlot.Date, enrichedSlot, booking, req.CustomOption); err != nil {
		log.Printf("[BookSlot] Error booking slot: %v", err)
		return nil, err
	}

	publicData := models.ToPublicBookingData(*booking)

	log.Printf("[BookSlot] Booking successful. ID: %s", booking.ID)
	return &publicData, nil
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
