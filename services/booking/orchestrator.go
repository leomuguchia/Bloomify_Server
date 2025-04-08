package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
)

// InitiateSession creates a new booking session.
func (s *DefaultBookingSessionService) InitiateSession(plan models.ServicePlan, userID, deviceID, userAgent string) (string, []models.ProviderDTO, error) {
	if err := validateServicePlan(plan); err != nil {
		log.Printf("ServicePlan validation error: %v", err)
		return "", nil, err
	}

	ctx := context.Background()
	sessionID := uuid.New().String()

	matchedProviders, err := s.MatchingSvc.MatchProviders(plan)
	if err != nil {
		log.Printf("Error matching providers: %v", err)
		return "", nil, fmt.Errorf("failed to match providers: %w", err)
	}

	if len(matchedProviders) == 0 {
		return "", nil, NewMatchError("no providers found matching criteria")
	}

	session := models.BookingSession{
		SessionID:           sessionID,
		ServicePlan:         plan,
		MatchedProviders:    matchedProviders,
		UserID:              userID,
		DeviceID:            deviceID,
		DeviceName:          userAgent,
		FullTimeSlotMapping: make(map[string]models.TimeSlot),
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("Error marshaling session: %v", err)
		return "", nil, fmt.Errorf("failed to marshal booking session: %w", err)
	}

	cacheClient := utils.GetBookingCacheClient()
	if err := cacheClient.Set(ctx, sessionID, sessionData, 30*time.Minute).Err(); err != nil {
		log.Printf("Error storing session in cache: %v", err)
		return "", nil, fmt.Errorf("failed to store booking session: %w", err)
	}

	log.Printf("Successfully initiated session: %s", sessionID)
	return sessionID, matchedProviders, nil
}

// UpdateSession retrieves the booking session from cache, validates the selected provider,
// computes weekly availability, and updates the session.
func (s *DefaultBookingSessionService) UpdateSession(sessionID string, selectedProviderID string, weekIndex int) (*models.BookingSession, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	if sessionID == "" {
		return nil, fmt.Errorf("booking session not initialized")
	}

	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired")
	}

	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	var selectedDTO models.ProviderDTO
	found := false
	for _, p := range session.MatchedProviders {
		if p.ID == selectedProviderID {
			selectedDTO = p
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("selected provider is not in the matched providers list")
	}

	session.SelectedProvider = selectedProviderID
	session.Availability = nil
	session.FullTimeSlotMapping = make(map[string]models.TimeSlot)

	selectedProvider := models.Provider{
		ID:               selectedDTO.ID,
		ServiceCatalogue: selectedDTO.ServiceCatalogue,
		Profile:          selectedDTO.Profile,
	}

	availabilityResult, err := s.SchedulerEngine.GetWeeklyAvailableSlots(selectedProvider, weekIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to compute availability for provider: %w", err)
	}

	if len(availabilityResult.Slots) == 0 {
		session.AvailabilityError = availabilityResult.AvailabilityError
	} else {
		session.Availability = availabilityResult.Slots
		session.FullTimeSlotMapping = availabilityResult.Mapping
		session.MaxAvailableDate = availabilityResult.MaxAvailableDate
	}

	updatedData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated booking session: %w", err)
	}
	if err := cacheClient.Set(ctx, sessionID, updatedData, 30*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to update booking session in cache: %w", err)
	}

	log.Printf("Successfully updated booking session: %s", sessionID)
	return &session, nil
}

func (s *DefaultBookingSessionService) ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlotResponse) (*models.Booking, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	// Retrieve the booking session from cache.
	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired")
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// Map the confirmed available slot's ID to the full TimeSlot.
	fullSlot, ok := session.FullTimeSlotMapping[confirmedSlot.ID]
	if !ok {
		return nil, fmt.Errorf("full timeslot not found for available slot %s", confirmedSlot.ID)
	}

	log.Printf("ConfirmBooking: Confirmed slot ID: %s, Full slot: Start %d, End %d", confirmedSlot.ID, fullSlot.Start, fullSlot.End)

	// Locate the selected provider.
	var selectedDTO models.ProviderDTO
	found := false
	for _, p := range session.MatchedProviders {
		if p.ID == session.SelectedProvider {
			selectedDTO = p
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("selected provider not found in booking session")
	}

	selectedProvider := models.Provider{
		ID:               selectedDTO.ID,
		ServiceCatalogue: selectedDTO.ServiceCatalogue,
		Profile:          selectedDTO.Profile,
	}

	// Build the BookingRequest using fields from the confirmed slot response.
	req := models.BookingRequest{
		ProviderID:    selectedProvider.ID,
		UserID:        session.UserID,
		Date:          confirmedSlot.Date,  // Use the date from the confirmed slot.
		Start:         confirmedSlot.Start, // Use the start time from the confirmed slot.
		End:           confirmedSlot.End,   // Use the end time from the confirmed slot.
		Units:         confirmedSlot.Units, // Use the units sent from the frontend.
		Priority:      false,
		PaymentMethod: "inApp",
	}

	// Process the booking using the SchedulerEngine.
	// Note: For one-off bookings, the result should be a models.Booking.
	result, err := s.SchedulerEngine.BookSlot(selectedProvider, req)
	if err != nil {
		return nil, fmt.Errorf("failed to book slot: %w", err)
	}

	booking, ok := result.(models.Booking)
	if !ok {
		return nil, fmt.Errorf("unexpected booking result type")
	}

	// Clear the session after a successful booking.
	cacheClient.Del(ctx, sessionID)

	return &booking, nil
}
