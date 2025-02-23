package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/services"
	"bloomify/utils"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BookServiceInput is the unified input for the integrated booking engine.
type BookServiceInput struct {
	// If empty, a new session is started (matching phase).
	SessionID string `json:"sessionID"`
	// The service plan details used for matching.
	ServicePlan models.ServicePlan `json:"servicePlan"`
	// The booking request details (provider selection, date, start time, etc.)
	BookingRequest services.BookingRequest `json:"bookingRequest"`
	// Optional: if the user confirmed a time slot.
	ConfirmedSlot *models.AvailableInterval `json:"confirmedSlot"`
	UserID        uint                      `json:"userID"`
}

// Resolver holds dependencies for GraphQL resolvers.
type Resolver struct {
	MatchingService services.MatchingService
	BookingService  services.BookingService
	CacheClient     *redis.Client
}

// BookService integrates matching, provider selection, availability, and booking confirmation.
func (r *Resolver) BookService(ctx context.Context, input BookServiceInput) (*models.BookingResponse, error) {
	logger := utils.GetLogger()

	// Create an empty response.
	resp := &models.BookingResponse{}

	// STEP 1: If no sessionID is provided, start a new session (matching phase).
	if input.SessionID == "" {
		// Run the matching phase using the provided service plan.
		matched, err := r.MatchingService.MatchProviders(input.ServicePlan)
		if err != nil {
			logger.Error("Matching failed", zap.Error(err))
			return nil, fmt.Errorf("matching failed: %w", err)
		}
		// Create a new session with the service plan and matched providers.
		session := models.BookingSession{
			ServicePlan:      input.ServicePlan,
			MatchedProviders: matched,
		}
		sessionID := uuid.New().String()
		sessionData, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal booking session: %w", err)
		}
		// Cache session for 10 minutes.
		if err := r.CacheClient.Set(ctx, sessionID, sessionData, 10*time.Minute).Err(); err != nil {
			return nil, fmt.Errorf("failed to cache booking session: %w", err)
		}
		// Set response with session context.
		resp.SessionID = sessionID
		resp.Providers = session.MatchedProviders
		return resp, nil
	}

	// STEP 2: Load existing session.
	sessionData, err := r.CacheClient.Get(ctx, input.SessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired")
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// STEP 3: If a new provider is selected, update the session.
	if input.BookingRequest.ProviderID != "" && session.SelectedProvider != input.BookingRequest.ProviderID {
		session.SelectedProvider = input.BookingRequest.ProviderID
		// Re-calculate availability for the selected provider.
		avail, err := r.BookingService.CheckAvailability(services.BookingRequest{
			ProviderID: input.BookingRequest.ProviderID,
			Date:       input.BookingRequest.Date,
			Duration:   input.BookingRequest.Duration,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve availability: %w", err)
		}
		session.Availability = avail
		// Update session in cache.
		updated, _ := json.Marshal(session)
		r.CacheClient.Set(ctx, input.SessionID, updated, 10*time.Minute)
		// Return updated session information.
		resp.SessionID = input.SessionID
		resp.Availability = session.Availability
		resp.Providers = session.MatchedProviders
		return resp, nil
	}

	// STEP 4: If a confirmed slot is provided, finalize the booking.
	if input.ConfirmedSlot != nil {
		bookingReq := services.BookingRequest{
			ProviderID:  session.SelectedProvider,
			UserID:      input.UserID,
			Date:        input.BookingRequest.Date,
			StartMinute: input.ConfirmedSlot.Start,
			Duration:    input.BookingRequest.Duration,
			Units:       input.BookingRequest.Units,
		}
		confirmedBooking, err := r.BookingService.BookSlot(bookingReq)
		if err != nil {
			logger.Error("Booking finalization failed", zap.Error(err))
			return nil, fmt.Errorf("booking finalization failed: %w", err)
		}
		// Clear the session from cache.
		r.CacheClient.Del(ctx, input.SessionID)
		// Return the confirmed booking.
		resp.Booking = confirmedBooking
		return resp, nil
	}

	return nil, fmt.Errorf("insufficient booking data provided")
}
