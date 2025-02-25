package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bloomify/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// BookingSessionService defines the interface for managing a stateful booking session.
type BookingSessionService interface {
	InitiateSession(plan models.ServicePlan) (string, []models.Provider, error)
	UpdateSession(sessionID string, selectedProviderID string) (*models.BookingSession, error)
	ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlot) (*models.Booking, error)
}

// DefaultBookingSessionService is our production implementation.
type DefaultBookingSessionService struct {
	MatchingSvc     MatchingService     // Matches providers based on the service plan.
	SchedulerEngine SchedulingEngine    // Computes available time slots.
	BookingSvc      BookingConfirmation // Finalizes booking creation.
	CacheClient     *redis.Client       // Redis client for session storage.
}

// InitiateSession creates a new booking session, assigns it a unique SessionID,
// and stores it in Redis. It returns the SessionID along with matched providers.
func (s *DefaultBookingSessionService) InitiateSession(plan models.ServicePlan) (string, []models.Provider, error) {
	ctx := context.Background()
	sessionID := uuid.New().String()

	matchedProviders, err := s.MatchingSvc.MatchProviders(plan)
	if err != nil {
		return "", nil, fmt.Errorf("failed to match providers: %w", err)
	}

	session := models.BookingSession{
		SessionID:        sessionID,
		ServicePlan:      plan,
		MatchedProviders: matchedProviders,
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal booking session: %w", err)
	}
	if err := s.CacheClient.Set(ctx, sessionID, sessionData, 10*time.Minute).Err(); err != nil {
		return "", nil, fmt.Errorf("failed to store booking session: %w", err)
	}

	return sessionID, matchedProviders, nil
}

// UpdateSession retrieves the session, updates it with the selected provider,
// computes available time slots using the scheduler engine, and saves the updated session.
func (s *DefaultBookingSessionService) UpdateSession(sessionID string, selectedProviderID string) (*models.BookingSession, error) {
	ctx := context.Background()

	sessionData, err := s.CacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	var selectedProvider models.Provider
	found := false
	for _, p := range session.MatchedProviders {
		if p.ID == selectedProviderID {
			selectedProvider = p
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("selected provider (%s) is not in the matched providers list", selectedProviderID)
	}

	session.SelectedProvider = selectedProviderID
	session.Availability = nil

	slots, err := s.SchedulerEngine.GetAvailableTimeSlots(selectedProvider, session.ServicePlan.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to compute availability: %w", err)
	}
	session.Availability = slots

	updatedData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated booking session: %w", err)
	}
	if err := s.CacheClient.Set(ctx, sessionID, updatedData, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to update booking session in cache: %w", err)
	}

	return &session, nil
}

// ConfirmBooking finalizes the booking by retrieving the session, constructing a Booking
// record using the confirmed available slot, creating the booking via BookingSvc,
// and then deleting the session.
func (s *DefaultBookingSessionService) ConfirmBooking(sessionID string, confirmedSlot models.AvailableSlot) (*models.Booking, error) {
	ctx := context.Background()

	sessionData, err := s.CacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	if session.SelectedProvider == "" {
		return nil, fmt.Errorf("no provider selected in booking session")
	}

	finalBooking := models.Booking{
		ProviderID: session.SelectedProvider,
		Date:       session.ServicePlan.Date,
		TimeSlot:   session.Availability[0], // Assumes client selected the first available slot.
		Units:      session.ServicePlan.Quantity,
		Status:     "Confirmed",
		CreatedAt:  time.Now(),
	}

	booking, err := s.BookingSvc.BookSlot(finalBooking)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize booking: %w", err)
	}

	s.CacheClient.Del(ctx, sessionID)

	return booking, nil
}
