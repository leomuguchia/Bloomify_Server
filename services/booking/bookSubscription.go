package booking

import (
	"bloomify/models"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// bookSubscriptionSlots creates recurring bookings over a subscription period.
// It returns only the first successfully booked instance for user confirmation.
func (se *DefaultSchedulingEngine) bookSubscriptionSlots(
	provider models.Provider,
	baseBooking models.Booking,
	subDetails models.SubscriptionDetails,
) (*models.PublicBookingData, error) {
	if subDetails.EndDate.Before(subDetails.StartDate) {
		return nil, fmt.Errorf("subscription end date is before start date")
	}

	totalDays := int(subDetails.EndDate.Sub(subDetails.StartDate).Hours()/24) + 1
	var firstBooking *models.Booking
	var once sync.Once
	var wg sync.WaitGroup
	errCh := make(chan error, totalDays)

	for d := 0; d < totalDays; d++ {
		current := subDetails.StartDate.AddDate(0, 0, d)
		wd := current.Weekday().String()

		switch subDetails.PlanType {
		case "daily":
			if contains(subDetails.ExemptedDays, wd) {
				continue
			}
		case "weekly":
			if wd != subDetails.Weekday {
				continue
			}
		default:
			continue
		}

		wg.Add(1)
		go func(bookDate time.Time) {
			defer wg.Done()
			dateStr := bookDate.Format("2006-01-02")

			const maxRetries = 3
			var selectedSlot *models.TimeSlot
			var err error

			for attempt := 1; attempt <= maxRetries; attempt++ {
				daySlots, fetchErr := se.TimeslotsRepo.GetAvailableTimeSlots(provider.ID, dateStr)
				if fetchErr != nil {
					err = fmt.Errorf("fetch error on %s: %w", dateStr, fetchErr)
					break
				}

				for _, ts := range daySlots {
					if ts.Start == baseBooking.Start && ts.End == baseBooking.End {
						selectedSlot = &ts
						break
					}
				}

				if selectedSlot == nil {
					err = fmt.Errorf("no matching slot [%d,%d] on %s", baseBooking.Start, baseBooking.End, dateStr)
					break
				}

				newB := baseBooking
				newB.ID = uuid.New().String()
				newB.Date = dateStr
				newB.CreatedAt = time.Now()

				err = se.bookSingleSlot(provider, dateStr, *selectedSlot, &newB, baseBooking.CustomOption)
				if err == nil {
					// Store only the first successful booking
					once.Do(func() {
						firstBooking = &newB
					})
					return // success
				}

				err = fmt.Errorf("attempt %d failed on %s: %w", attempt, dateStr, err)
				time.Sleep(1 * time.Second)
			}

			if err != nil {
				errCh <- err
			}
		}(current)
	}

	wg.Wait()
	close(errCh)

	if firstBooking == nil {
		return nil, fmt.Errorf("subscription booking failed: no successful booking")
	}

	publicBooking := models.ToPublicBookingData(*firstBooking)
	return &publicBooking, nil
}
