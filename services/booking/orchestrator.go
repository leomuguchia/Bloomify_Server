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
	// Set TTL to 30 minutes
	if err := cacheClient.Set(ctx, sessionID, sessionData, 30*time.Minute).Err(); err != nil {
		log.Printf("Error storing session in cache: %v", err)
		return "", nil, fmt.Errorf("failed to store booking session: %w", err)
	}

	log.Printf("Successfully initiated session: %s", sessionID)
	return sessionID, matchedProviders, nil
}

// UpdateSession updates the session with the selected provider.
func (s *DefaultBookingSessionService) UpdateSession(sessionID string, selectedProviderID string) (*models.BookingSession, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	log.Printf("UpdateSession: Starting update for sessionID: %s with providerID: %s", sessionID, selectedProviderID)

	if sessionID == "" {
		errMsg := "booking not initialized"
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Retrieve session data from cache.
	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		errMsg := fmt.Sprintf("booking session not found or expired %s", err)
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}
	log.Printf("UpdateSession: Retrieved session data for sessionID: %s", sessionID)

	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		errMsg := fmt.Sprintf("failed to parse booking session for sessionID %s: %v", sessionID, err)
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}
	log.Printf("UpdateSession: Unmarshaled session data successfully for sessionID: %s", sessionID)

	// Validate that the selected provider exists in the matched providers list.
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
		errMsg := fmt.Sprintf("selected provider (%s) is not in the matched providers list", selectedProviderID)
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}
	log.Printf("UpdateSession: Found selected provider with ID: %s", selectedProviderID)

	// Update session with the selected provider.
	session.SelectedProvider = selectedProviderID
	session.Availability = nil
	session.FullTimeSlotMapping = make(map[string]models.TimeSlot)
	log.Printf("UpdateSession: Cleared availability and full time slot mapping for sessionID: %s", sessionID)

	// Convert provider DTO to minimal Provider.
	selectedProvider := models.Provider{
		ID:               selectedDTO.ID,
		ServiceCatalogue: selectedDTO.ServiceCatalogue,
		Profile:          selectedDTO.Profile,
	}
	log.Printf("UpdateSession: Converted provider DTO to minimal provider for providerID: %s", selectedProvider.ID)

	// Get available slots and mapping from the scheduler.
	log.Printf("UpdateSession: Calling GetAvailableTimeSlots for providerID: %s", selectedProvider.ID)
	availabilityResult, err := s.SchedulerEngine.GetAvailableTimeSlots(selectedProvider, 0)
	if err != nil {
		log.Printf("UpdateSession: failed to compute availability for provider")
		return nil, fmt.Errorf("failed to compute availability for provider")
	}

	log.Printf("UpdateSession: finished calling GetAvailableTimeSlots for providerID: %s", selectedProvider.ID)
	// Validate the mapping keys match the expected available slot IDs.
	for key, slot := range availabilityResult.Mapping {
		log.Printf("UpdateSession: Scheduler mapping - Key: %s, Slot ID: %s, Start: %d, End: %d", key, slot.ID, slot.Start, slot.End)
		if key != slot.ID {
			log.Printf("WARNING: Mapping key (%s) does not match slot ID (%s). Please verify scheduler contract.", key, slot.ID)
		}
	}

	session.Availability = availabilityResult.Slots
	session.FullTimeSlotMapping = availabilityResult.Mapping
	log.Printf("UpdateSession: Computed availability with %d slots for providerID: %s", len(availabilityResult.Slots), selectedProvider.ID)

	// Marshal the updated session.
	updatedData, err := json.Marshal(session)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal updated booking session for sessionID %s: %v", sessionID, err)
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Save the updated session back into cache with a TTL of 30 minutes.
	if err := cacheClient.Set(ctx, sessionID, updatedData, 30*time.Minute).Err(); err != nil {
		errMsg := fmt.Sprintf("failed to update booking session in cache for sessionID %s: %v", sessionID, err)
		log.Printf("UpdateSession: %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}
	log.Printf("UpdateSession: Successfully updated booking session in cache for sessionID: %s", sessionID)

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

	// Validate that the confirmed slot's ID is indeed expected.
	log.Printf("ConfirmBooking: Confirmed slot ID: %s, Full slot details: Start %d, End %d", confirmedSlot.ID, fullSlot.Start, fullSlot.End)

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
