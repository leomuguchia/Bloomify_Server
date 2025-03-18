// File: booking/booking_session_service.go
package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
)

// InitiateSession creates a new booking session, assigns it a unique SessionID,
// and stores it in Redis. It returns the SessionID along with matched providers (as DTOs).
func (s *DefaultBookingSessionService) InitiateSession(plan models.ServicePlan, userID, deviceID, userAgent string) (string, []models.ProviderDTO, error) {
	ctx := context.Background()
	sessionID := uuid.New().String()

	matchedProviders, err := s.MatchingSvc.MatchProviders(plan)
	if err != nil { // ✅ Correct error handling
		return "", nil, fmt.Errorf("failed to match providers: %w", err)
	}

	session := models.BookingSession{
		SessionID:        sessionID,
		ServicePlan:      plan,
		MatchedProviders: matchedProviders,
		UserID:           userID,
		DeviceID:         deviceID,
		UserAgent:        userAgent,
	}

	sessionData, err := json.Marshal(session)
	if err != nil { // ✅ Correct error check
		return "", nil, fmt.Errorf("failed to marshal booking session: %w", err)
	}

	cacheClient := utils.GetBookingCacheClient()
	if err := cacheClient.Set(ctx, sessionID, sessionData, 10*time.Minute).Err(); err != nil { // ✅ Correct
		return "", nil, fmt.Errorf("failed to store booking session: %w", err)
	}

	return sessionID, matchedProviders, nil
}

// UpdateSession retrieves the session, updates it with the selected provider,
// computes available time slots using the scheduler engine, and saves the updated session.
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

	// Verify that the selected provider is among the matched providers (DTOs).
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

	// Convert selectedDTO to a minimal Provider (only required fields for scheduling).
	selectedProvider := models.Provider{
		ID:          selectedDTO.ID,
		ServiceType: selectedDTO.ServiceType,
		Location:    selectedDTO.Location,
		LocationGeo: selectedDTO.LocationGeo,
		Profile:     selectedDTO.Profile,
	}

	slots, err := s.SchedulerEngine.GetAvailableTimeSlots(selectedProvider, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compute availability: %w", err)
	}
	session.Availability = slots

	updatedData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated booking session: %w", err)
	}
	if err := cacheClient.Set(ctx, sessionID, updatedData, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to update booking session in cache: %w", err)
	}

	return &session, nil
}

// ConfirmBooking finalizes the booking by retrieving the session from Redis,
// and then calling the scheduler's BookSlot method to reserve the confirmed slot.
func (s *DefaultBookingSessionService) ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlot) (*models.Booking, error) {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()

	// Retrieve the booking session.
	sessionData, err := cacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// Locate the selected provider from the matched providers (DTOs).
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

	// Convert DTO to a minimal Provider.
	selectedProvider := models.Provider{
		ID:          selectedDTO.ID,
		ServiceType: selectedDTO.ServiceType,
		Location:    selectedDTO.Location,
		LocationGeo: selectedDTO.LocationGeo,
		Profile:     selectedDTO.Profile,
	}

	bookingRecord := models.Booking{
		ProviderID:   selectedProvider.ID,
		ProviderName: selectedProvider.Profile.ProviderName,
		UserID:       session.UserID,
		Date:         session.ServicePlan.Date,
		Start:        confirmedSlot.Start,
		End:          confirmedSlot.End,
		CreatedAt:    time.Now(),
	}

	if err := s.SchedulerEngine.BookSlot(selectedProvider, session.ServicePlan.Date, confirmedSlot, bookingRecord); err != nil {
		return nil, fmt.Errorf("failed to book slot: %w", err)
	}

	cacheClient.Del(ctx, sessionID)
	return &bookingRecord, nil
}

// CancelSession allows the client to explicitly cancel a booking session.
// It deletes the session data from the cache.
func (s *DefaultBookingSessionService) CancelSession(sessionID string) error {
	ctx := context.Background()
	cacheClient := utils.GetBookingCacheClient()
	if err := cacheClient.Del(ctx, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to cancel booking session: %w", err)
	}
	return nil
}
