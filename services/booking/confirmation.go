package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bloomify/database/repository"
	"bloomify/models"

	"github.com/go-redis/redis/v8"
)

// BookingConfirmation defines methods to finalize a booking.
type BookingConfirmation interface {
	Confirm(sessionID string) (*models.BookingConfirmationResponse, error)
}

// DefaultBookingConfirmation implements BookingConfirmation.
type DefaultBookingConfirmation struct {
	Repo        repository.BookingRepository // Repository for persisting bookings
	CacheClient *redis.Client                // Redis client for session retrieval
}

// Confirm retrieves the booking session, re-checks availability (including priority capacity if needed),
// finalizes the booking, cleans up the session, and returns a confirmation response.
func (bc *DefaultBookingConfirmation) Confirm(sessionID string) (*models.BookingConfirmationResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Retrieve the session from Redis.
	sessionData, err := bc.CacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("booking session not found or expired: %w", err)
	}

	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse booking session: %w", err)
	}

	// Ensure a provider was selected.
	if session.SelectedProvider == "" {
		return nil, fmt.Errorf("no provider selected in booking session")
	}

	// For this example, we select the first available interval.
	if len(session.Availability) == 0 {
		return nil, fmt.Errorf("no available interval found in session")
	}
	selectedInterval := session.Availability[0]

	// Construct final booking details.
	// Depending on whether the booking is urgent (priority) or not, check capacity accordingly.
	var capacityAvailable int
	if session.ServicePlan.Priority {
		capacityAvailable = selectedInterval.PriorityCapacityRemaining
	} else {
		capacityAvailable = selectedInterval.RegularCapacityRemaining
	}
	// Ensure the requested units do not exceed available capacity.
	if session.ServicePlan.RequestedUnits > capacityAvailable {
		return nil, fmt.Errorf("requested units (%d) exceed available capacity (%d) for the selected interval",
			session.ServicePlan.RequestedUnits, capacityAvailable)
	}

	finalBooking := models.Booking{
		ProviderID:  session.SelectedProvider,
		Date:        session.ServicePlan.Date,
		StartMinute: selectedInterval.Start, // Use the interval start as the booking start.
		Duration:    session.ServicePlan.Duration,
		Units:       session.ServicePlan.RequestedUnits,
		// Populate additional booking fields as needed.
	}

	// Persist the final booking.
	if err := bc.Repo.CreateBooking(&finalBooking); err != nil {
		return nil, fmt.Errorf("failed to create final booking: %w", err)
	}

	// If the booking fills the capacity, record a block.
	// (The repository or scheduler engine might handle this normally; here we add it explicitly.)
	updatedUsage := capacityAvailable - capacityAvailable + session.ServicePlan.RequestedUnits // simplified for example
	if updatedUsage >= capacityAvailable {
		block := models.Blocked{
			ProviderID:  session.SelectedProvider,
			Date:        session.ServicePlan.Date,
			Start:       selectedInterval.Start,
			End:         selectedInterval.End,
			Reason:      "capacity reached",
			ServiceType: session.ServicePlan.ServiceType,
		}
		if err := bc.Repo.CreateBlockedInterval(&block); err != nil {
			// Log warning but do not fail the booking.
			fmt.Printf("warning: failed to create block: %v\n", err)
		}
	}

	// Delete the session from Redis.
	bc.CacheClient.Del(ctx, sessionID)

	// Build the confirmation response.
	response := models.BookingConfirmationResponse{
		BookingID:    finalBooking.ID,
		ProviderID:   finalBooking.ProviderID,
		Date:         finalBooking.Date,
		StartMinute:  finalBooking.StartMinute,
		Duration:     finalBooking.Duration,
		Confirmation: "Booking confirmed",
	}

	return &response, nil
}
