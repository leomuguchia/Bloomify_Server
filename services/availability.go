// services/availability.go
package services

import (
	"fmt"

	"bloomify/database"
	"bloomify/models"
)

// AvailabilityService defines methods for computing available intervals.
type AvailabilityService interface {
	GetAvailableIntervals(providerID, date string, requestedDuration int) ([]models.AvailableInterval, error)
}

// DefaultAvailabilityService is a concrete implementation.
type DefaultAvailabilityService struct{}

// GetAvailableIntervals computes the available time intervals for a provider on a given date
// that can accommodate the requested duration.
func (s *DefaultAvailabilityService) GetAvailableIntervals(providerID, date string, requestedDuration int) ([]models.AvailableInterval, error) {
	var provider models.Provider
	if err := database.DB.First(&provider, "id = ?", providerID).Error; err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Retrieve blocked intervals for the provider on the given date.
	var blocked []models.Blocked
	if err := database.DB.Where("provider_id = ? AND date = ?", providerID, date).Find(&blocked).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve blocked intervals: %w", err)
	}

	// Compute continuous available intervals from working hours minus blocked intervals.
	intervals := computeContinuousIntervals(
		struct{ Start, End int }{provider.WorkingStart, provider.WorkingEnd},
		blocked,
	)

	// Filter intervals by requested duration.
	var available []models.AvailableInterval
	for _, iv := range intervals {
		if iv.End-iv.Start >= requestedDuration {
			available = append(available, models.AvailableInterval{
				Start: iv.Start,
				End:   iv.End,
				Label: fmt.Sprintf("%s - %s", formatTime(iv.Start), formatTime(iv.End)),
			})
		}
	}

	return available, nil
}

// continuousInterval is a helper struct.
type continuousInterval struct {
	Start int
	End   int
}

// computeContinuousIntervals subtracts blocked intervals from working hours.
func computeContinuousIntervals(working struct{ Start, End int }, blocked []models.Blocked) []continuousInterval {
	available := []continuousInterval{{Start: working.Start, End: working.End}}
	for _, block := range blocked {
		var updated []continuousInterval
		for _, iv := range available {
			if block.EndMinute <= iv.Start || block.StartMinute >= iv.End {
				updated = append(updated, iv)
				continue
			}
			if block.StartMinute > iv.Start {
				updated = append(updated, continuousInterval{Start: iv.Start, End: block.StartMinute})
			}
			if block.EndMinute < iv.End {
				updated = append(updated, continuousInterval{Start: block.EndMinute, End: iv.End})
			}
		}
		available = updated
	}
	return available
}
