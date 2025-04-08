package booking

import (
	"fmt"
	"sync"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SubscriptionBookingResult aggregates successful bookings and errors for a subscription.
type SubscriptionBookingResult struct {
	SuccessfulBookings []models.Booking
	Errors             []error
}

// BookSlot processes both one-off and subscription bookings.
func (se *DefaultSchedulingEngine) BookSlot(provider models.Provider, req models.BookingRequest) (interface{}, error) {
	// Subscription booking branch.
	if req.Subscription != nil {
		baseBooking := models.Booking{
			ID:            uuid.New().String(),
			ProviderID:    provider.ID,
			UserID:        req.UserID,
			Units:         req.Units,
			Start:         req.Start,
			End:           req.End,
			UnitType:      provider.ServiceCatalogue.ServiceType,
			Priority:      req.Priority,
			PaymentMethod: req.PaymentMethod,
		}
		// Return the result containing successes and errors.
		return se.bookSubscriptionSlots(provider, baseBooking, *req.Subscription)
	}

	// One-off booking branch.
	if req.Date == "" || req.Start == 0 || req.End == 0 {
		return nil, fmt.Errorf("missing date or time details for one-off booking")
	}

	daySlots, err := se.Repo.GetAvailableTimeSlots(provider.ID, req.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeslots for date %s: %w", req.Date, err)
	}
	if len(daySlots) == 0 {
		return nil, fmt.Errorf("no available timeslots for date %s", req.Date)
	}

	var selectedSlot *models.TimeSlot
	for _, ts := range daySlots {
		if ts.Start == req.Start && ts.End == req.End {
			selectedSlot = &ts
			break
		}
	}
	if selectedSlot == nil {
		return nil, fmt.Errorf("no matching timeslot available for requested time [%d, %d] on %s", req.Start, req.End, req.Date)
	}

	booking := models.Booking{
		ID:            uuid.New().String(),
		ProviderID:    provider.ID,
		UserID:        req.UserID,
		Date:          req.Date,
		Start:         req.Start,
		End:           req.End,
		Units:         req.Units,
		UnitType:      provider.ServiceCatalogue.ServiceType,
		Priority:      req.Priority,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     time.Now(),
	}

	err = se.bookSingleSlot(provider, req.Date, *selectedSlot, booking, req.CustomOption)
	if err != nil {
		return nil, err
	}
	// For one-off, return the successful booking record.
	return booking, nil
}

// bookSubscriptionSlots processes recurring bookings over a subscription period.
// It returns a SubscriptionBookingResult containing the list of successful bookings and any errors.
func (se *DefaultSchedulingEngine) bookSubscriptionSlots(provider models.Provider, baseBooking models.Booking, subDetails models.SubscriptionDetails) (*SubscriptionBookingResult, error) {
	if subDetails.EndDate.Before(subDetails.StartDate) {
		return nil, fmt.Errorf("subscription end date is before start date")
	}
	totalDays := int(subDetails.EndDate.Sub(subDetails.StartDate).Hours()/24) + 1
	// Use provider-defined active days.
	activeDays := provider.SubscriptionModel.ActiveDays

	var wg sync.WaitGroup
	result := &SubscriptionBookingResult{
		SuccessfulBookings: make([]models.Booking, 0),
		Errors:             make([]error, 0),
	}
	errCh := make(chan dayBookingResult, totalDays)

	// Batch process each day concurrently.
	for d := 0; d < totalDays; d++ {
		currentDay := subDetails.StartDate.AddDate(0, 0, d)
		weekday := currentDay.Weekday().String()
		if !contains(activeDays, weekday) {
			continue
		}

		wg.Add(1)
		go func(day time.Time) {
			defer wg.Done()
			dateStr := day.Format("2006-01-02")
			var bookingResult dayBookingResult
			bookingResult.date = dateStr

			// Retry mechanism: attempt up to maxRetries.
			const maxRetries = 3
			var err error
			var selectedSlot *models.TimeSlot
			for attempt := 1; attempt <= maxRetries; attempt++ {
				daySlots, fetchErr := se.Repo.GetAvailableTimeSlots(provider.ID, dateStr)
				if fetchErr != nil {
					err = fmt.Errorf("error fetching timeslots for date %s: %w", dateStr, fetchErr)
				} else {
					// Look for a matching slot.
					for _, ts := range daySlots {
						if ts.Start == baseBooking.Start && ts.End == baseBooking.End {
							selectedSlot = &ts
							break
						}
					}
					if selectedSlot == nil {
						err = fmt.Errorf("no matching timeslot (from %d to %d) available on %s", baseBooking.Start, baseBooking.End, dateStr)
					} else {
						// Attempt booking.
						newBooking := baseBooking
						newBooking.ID = uuid.New().String()
						newBooking.Date = dateStr
						newBooking.CreatedAt = time.Now()
						newBooking.Start = selectedSlot.Start
						newBooking.End = selectedSlot.End

						err = se.bookSingleSlot(provider, dateStr, *selectedSlot, newBooking, nil)
						if err == nil {
							bookingResult.booking = newBooking
							break // Booking succeeded.
						} else {
							err = fmt.Errorf("attempt %d: failed to book subscription slot for %s: %w", attempt, dateStr, err)
						}
					}
				}
				// Wait a moment before next retry.
				time.Sleep(1 * time.Second)
			}
			bookingResult.err = err
			errCh <- bookingResult
		}(currentDay)
	}

	wg.Wait()
	close(errCh)
	// Collect results.
	for res := range errCh {
		if res.err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("date %s: %w", res.date, res.err))
		} else {
			result.SuccessfulBookings = append(result.SuccessfulBookings, res.booking)
		}
	}
	return result, nil
}

// dayBookingResult is used internally to pass the result of each day’s booking attempt.
type dayBookingResult struct {
	date    string
	booking models.Booking
	err     error
}

func (se *DefaultSchedulingEngine) bookSingleSlot(provider models.Provider, date string, slot models.TimeSlot, booking models.Booking, customOption *models.CustomOption) error {
	// Validate that the booking's time falls within the slot boundaries.
	if booking.Start < slot.Start || booking.End > slot.End {
		return fmt.Errorf("booking time [%d, %d] is not within timeslot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	// Validate and recalculate pricing.
	confirmation, err := ValidateAndBook(provider.ID, slot, booking, provider.ServiceCatalogue, customOption)
	if err != nil {
		return fmt.Errorf("provider booking validation failed: %w", err)
	}
	booking.ProviderID = provider.ID
	booking.Date = date
	booking.CreatedAt = time.Now()
	booking.TotalPrice = confirmation.TotalPrice
	// Store the timeslot ID in the booking for synchronization.
	booking.TimeSlotID = slot.ID

	// Insert the full booking document into the bookings collection.
	if err := se.Repo.CreateBooking(&booking); err != nil {
		return fmt.Errorf("error creating booking: %w", err)
	}

	// Use the repository method to embed the booking reference
	// into the provider's embedded timeslot (and update aggregate counters).
	if err := se.Repo.EmbedBookingReference(provider.ID, slot.ID, date, booking.ID, booking.Units, booking.Priority); err != nil {
		// If we fail to embed the booking reference, cancel the booking.
		_ = se.Repo.CancelBooking(booking.ID)
		return fmt.Errorf("failed to embed booking reference: %w", err)
	}

	// Re-check the current usage (aggregate) for the slot.
	updatedUsage, err := se.Repo.SumOverlappingBookings(provider.ID, date, slot.Start, slot.End)
	if err != nil {
		return fmt.Errorf("failed to re-check capacity: %w", err)
	}

	// If the updated usage meets or exceeds capacity, update the timeslot's blocked flag.
	if updatedUsage >= slot.Capacity {
		// In our hybrid approach, we update the embedded timeslot's Blocked flag and optional reason.
		if err := se.Repo.SetEmbeddedTimeSlotBlocked(provider.ID, slot.ID, date, true, "capacity reached"); err != nil {
			// Log the error but do not necessarily fail the booking.
			utils.GetLogger().Error("failed to mark timeslot as blocked", zap.String("slotID", slot.ID), zap.Error(err))
		}
	}

	// Process payment if pre-payment is required.
	if provider.PaymentDetails.PrePaymentRequired {
		invoiceCh := make(chan *models.Invoice)
		errCh := make(chan error)
		go func(b models.Booking) {
			invoice, payErr := se.PaymentHandler.ProcessPayment(&b)
			if payErr != nil {
				errCh <- payErr
				return
			}
			invoiceCh <- invoice
		}(booking)

		select {
		case invoice := <-invoiceCh:
			if invoice.Status != "paid" {
				// Payment not confirmed: cancel booking and roll back embedded updates.
				_ = se.Repo.CancelBooking(booking.ID)
				rollbackErr := se.Repo.RollbackEmbeddedTimeSlotAggregates(provider.ID, slot.ID, date, booking.Units, booking.Priority, slot.Version)
				if rollbackErr != nil {
					utils.GetLogger().Warn("failed to rollback aggregates for booking", zap.String("bookingID", booking.ID), zap.Error(rollbackErr))
				}
				return fmt.Errorf("payment not confirmed: invoice status %s", invoice.Status)
			}
		case err := <-errCh:
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackEmbeddedTimeSlotAggregates(provider.ID, slot.ID, date, booking.Units, booking.Priority, slot.Version)
			if rollbackErr != nil {
				utils.GetLogger().Warn("failed to rollback aggregates for booking", zap.String("bookingID", booking.ID), zap.Error(rollbackErr))
			}
			return fmt.Errorf("payment processing error: %w", err)
		case <-time.After(5 * time.Minute):
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackEmbeddedTimeSlotAggregates(provider.ID, slot.ID, date, booking.Units, booking.Priority, slot.Version)
			if rollbackErr != nil {
				utils.GetLogger().Warn("failed to rollback aggregates for booking", zap.String("bookingID", booking.ID), zap.Error(rollbackErr))
			}
			return fmt.Errorf("payment processing timed out; booking cancelled")
		}
	} else {
		booking.PaymentMethod = "cash-on-service"
		booking.PaymentStatus = "pending"
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
