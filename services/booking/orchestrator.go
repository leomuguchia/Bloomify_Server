package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"log"

	"github.com/google/uuid"
)

// InitiateSession creates a new booking session.
func (s *DefaultBookingSessionService) InitiateSession(plan models.ServicePlan, userID, deviceID, userAgent string) (string, []models.ProviderDTO, error) {
	ctx := context.Background()
	sessionID := uuid.New().String()

	matchedProviders, err := s.MatchingSvc.MatchProviders(plan)
	if err != nil {
		log.Printf("Error matching providers: %v", err)
		return "", nil, fmt.Errorf("failed to match providers: %w", err)
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
	if err := cacheClient.Set(ctx, sessionID, sessionData, 10*time.Minute).Err(); err != nil {
		log.Printf("Error storing session in cache: %v", err)
		return "", nil, fmt.Errorf("failed to store booking session: %w", err)
	}

	return sessionID, matchedProviders, nil
}

// UpdateSession updates the session with the selected provider.
func (s *DefaultBookingSessionService) UpdateSession(sessionID string, selectedProviderID string) (*models.BookingSession, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// Validate that the selected provider exists.
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
		return nil, fmt.Errorf("selected provider (%s) is not in the matched providers list", selectedProviderID)
	}

	session.SelectedProvider = selectedProviderID
	session.Availability = nil
	session.FullTimeSlotMapping = make(map[string]models.TimeSlot)

	// Convert provider DTO to minimal Provider.
	selectedProvider := models.Provider{
		ID:               selectedDTO.ID,
		ServiceCatalogue: selectedDTO.ServiceCatalogue,
		LocationGeo:      selectedDTO.LocationGeo,
		Profile:          selectedDTO.Profile,
	}

	// Get available slots and mapping.
	availabilityResult, err := s.SchedulerEngine.GetAvailableTimeSlots(selectedProvider, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compute availability: %w", err)
	}
	session.Availability = availabilityResult.Slots
	session.FullTimeSlotMapping = availabilityResult.Mapping

	updatedData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated booking session: %w", err)
	}
	if err := cacheClient.Set(ctx, sessionID, updatedData, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to update booking session in cache: %w", err)
	}

	return &session, nil
}

// ConfirmBooking finalizes the booking by mapping the confirmed AvailableSlot to its full TimeSlot.
func (s *DefaultBookingSessionService) ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlot) (*models.Booking, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// Retrieve the full TimeSlot using the mapping.
	fullSlot, ok := session.FullTimeSlotMapping[confirmedSlot.ID]
	if !ok {
		return nil, fmt.Errorf("full timeslot not found for available slot %s", confirmedSlot.ID)
	}

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

	// Convert DTO to minimal Provider.
	selectedProvider := models.Provider{
		ID:               selectedDTO.ID,
		ServiceCatalogue: selectedDTO.ServiceCatalogue,
		LocationGeo:      selectedDTO.LocationGeo,
		Profile:          selectedDTO.Profile,
	}

	// Build the booking record.
	bookingRecord := models.Booking{
		ProviderID:   selectedProvider.ID,
		ProviderName: selectedProvider.Profile.ProviderName,
		UserID:       session.UserID,
		Date:         session.ServicePlan.Date,
		Start:        fullSlot.Start,
		End:          fullSlot.End,
		CreatedAt:    time.Now(),
	}

	// Call BookSlot with the full TimeSlot.
	if err := s.SchedulerEngine.BookSlot(selectedProvider, session.ServicePlan.Date, fullSlot, bookingRecord); err != nil {
		return nil, fmt.Errorf("failed to book slot: %w", err)
	}

	// Clear session after successful booking.
	cacheClient.Del(ctx, sessionID)
	return &bookingRecord, nil
}
