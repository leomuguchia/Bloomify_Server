package services

import (
	"fmt"
	"time"

	"bloomify/database"
	"bloomify/models"

	"gorm.io/gorm"
)

// BookingRequest holds all information required for a booking.
type BookingRequest struct {
	ProviderID  string `json:"provider_id"`  // Provider to book
	UserID      uint   `json:"user_id"`      // User making the booking
	Date        string `json:"date"`         // Booking date in YYYY-MM-DD
	StartMinute int    `json:"start_minute"` // Proposed start time (minutes from midnight)
	Duration    int    `json:"duration"`     // Duration of the booking (minutes)
	Units       int    `json:"units"`        // Number of capacity units requested
	// Optional: Urgency flag ("now" or "scheduled") could be added here.
}

// BookingService defines the interface for booking operations.
type BookingService interface {
	BookSlot(req BookingRequest) (*models.Booking, error)
	CheckAvailability(req BookingRequest) ([]models.AvailableInterval, error)
}

// DefaultBookingService is the default, robust implementation.
type DefaultBookingService struct {
	// In a full implementation, you could inject repository interfaces here.
	// For now, we use the global database connection.
}

// BookSlot validates a booking request, checks capacity, calculates pricing, and creates a booking atomically.
func (s *DefaultBookingService) BookSlot(req BookingRequest) (*models.Booking, error) {
	var createdBooking *models.Booking

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Retrieve provider details.
		var provider models.Provider
		if err := tx.First(&provider, "id = ?", req.ProviderID).Error; err != nil {
			return fmt.Errorf("provider not found: %w", err)
		}

		// 2. Validate the booking falls within the provider's working hours.
		if req.StartMinute < provider.WorkingStart || req.StartMinute+req.Duration > provider.WorkingEnd {
			return fmt.Errorf("requested time (%d-%d) is outside provider working hours (%d-%d)",
				req.StartMinute, req.StartMinute+req.Duration, provider.WorkingStart, provider.WorkingEnd)
		}

		// 3. Compute available intervals for the requested date.
		availIntervals, err := computeAvailability(tx, provider, req.Date)
		if err != nil {
			return fmt.Errorf("failed to compute availability: %w", err)
		}

		// 4. Verify that the requested slot fits completely within one available interval.
		if !isSlotWithinIntervals(availIntervals, req.StartMinute, req.Duration) {
			return fmt.Errorf("requested slot [%d, %d) is not available", req.StartMinute, req.StartMinute+req.Duration)
		}

		// 5. Check overlapping bookings to ensure sufficient capacity.
		var bookedUnits int64
		if err := tx.Table("bookings").
			Select("COALESCE(SUM(units), 0)").
			Where("provider_id = ? AND date = ? AND NOT (start_minute >= ? OR (start_minute + duration) <= ?)",
				req.ProviderID, req.Date, req.StartMinute+req.Duration, req.StartMinute).
			Scan(&bookedUnits).Error; err != nil {
			return fmt.Errorf("capacity check failed: %w", err)
		}
		remainingCapacity := provider.Capacity - int(bookedUnits)
		if remainingCapacity < req.Units {
			return fmt.Errorf("insufficient capacity: available %d, requested %d", remainingCapacity, req.Units)
		}

		// 6. Calculate total price based on the provider's pricing model.
		totalPrice := calculatePrice(provider, req.Duration)

		// 7. Create the booking record.
		newBooking := models.Booking{
			ProviderID:  req.ProviderID,
			UserID:      req.UserID,
			Date:        req.Date,
			StartMinute: req.StartMinute,
			Duration:    req.Duration,
			Units:       req.Units,
			TotalPrice:  totalPrice,
			Status:      "Confirmed",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := tx.Create(&newBooking).Error; err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}
		createdBooking = &newBooking
		return nil
	})

	if err != nil {
		return nil, err
	}
	return createdBooking, nil
}

// CheckAvailability returns the available intervals for the given booking request.
// This function is useful for suggesting alternative times when the requested slot isn't available.
func (s *DefaultBookingService) CheckAvailability(req BookingRequest) ([]models.AvailableInterval, error) {
	// Retrieve provider details.
	var provider models.Provider
	if err := database.DB.First(&provider, "id = ?", req.ProviderID).Error; err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}
	availIntervals, err := computeAvailability(database.DB, provider, req.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to compute availability: %w", err)
	}

	// Filter intervals to include only those long enough for the requested duration.
	var filtered []models.AvailableInterval
	for _, iv := range availIntervals {
		if iv.End-iv.Start >= req.Duration {
			filtered = append(filtered, iv)
		}
	}
	return filtered, nil
}

// computeAvailability retrieves blocked intervals for a provider on a given date and computes available intervals.
func computeAvailability(tx *gorm.DB, provider models.Provider, date string) ([]models.AvailableInterval, error) {
	var blocked []models.Blocked
	if err := tx.Where("provider_id = ? AND date = ?", provider.ID, date).Find(&blocked).Error; err != nil {
		return nil, err
	}
	intervals := computeContinuousIntervals(
		struct{ Start, End int }{provider.WorkingStart, provider.WorkingEnd},
		blocked,
	)
	var avail []models.AvailableInterval
	for _, iv := range intervals {
		avail = append(avail, models.AvailableInterval{
			Start: iv.Start,
			End:   iv.End,
			Label: fmt.Sprintf("%s - %s", formatTime(iv.Start), formatTime(iv.End)),
		})
	}
	return avail, nil
}

// isSlotWithinIntervals checks if the requested slot (from start to start+duration) is fully contained within one available interval.
func isSlotWithinIntervals(intervals []models.AvailableInterval, start, duration int) bool {
	end := start + duration
	for _, iv := range intervals {
		if start >= iv.Start && end <= iv.End {
			return true
		}
	}
	return false
}

// calculatePrice computes the total price based on the provider's pricing model.
func calculatePrice(provider models.Provider, duration int) float64 {
	switch provider.PricingModel {
	case "Hourly":
		hours := float64(duration) / 60.0
		return provider.BaseRate * hours
	case "PerUnit":
		return provider.BaseRate * float64(duration) / 60.0
	case "FlatRate":
		return provider.BaseRate
	default:
		return 0.0
	}
}

// formatTime converts minutes from midnight into a human-readable time string.
func formatTime(minutes int) string {
	hour := minutes / 60
	minute := minutes % 60
	ampm := "AM"
	if hour >= 12 {
		ampm = "PM"
	}
	hour = hour % 12
	if hour == 0 {
		hour = 12
	}
	return fmt.Sprintf("%d:%02d %s", hour, minute, ampm)
}
