package booking

import (
	"context"
	"fmt"
	"sort"
	"time"

	schedulerRepo "bloomify/database/repository/scheduler"
	"bloomify/models"
)

// SchedulingEngine defines methods to compute available time slots for a provider.
type SchedulingEngine interface {
	// GetAvailableTimeSlots computes available slots for a provider over a 7‑day booking window.
	GetAvailableTimeSlots(provider models.Provider, date string) ([]models.AvailableSlot, error)
	// BookSlot creates a booking record for a provider, given a selected available slot.
	BookSlot(provider models.Provider, date string, slot models.AvailableSlot, booking models.Booking) error
}

// DefaultSchedulingEngine is our production‑ready implementation.
type DefaultSchedulingEngine struct {
	Repo           schedulerRepo.SchedulerRepository
	PaymentHandler PaymentProcessor // In-app payment processor
}

// GetAvailableTimeSlots computes available time slots for the provider over a 7‑day window.
// It instantiates each recurring slot template for each day (attaching the date) and applies cutoff rules.
// It now uses denormalized aggregates stored on the TimeSlot to compute usage instead of heavy aggregation.
func (se *DefaultSchedulingEngine) GetAvailableTimeSlots(provider models.Provider, _ string) ([]models.AvailableSlot, error) {
	now := time.Now()
	// Define the booking window cutoff as 7 days from now.
	cutoff := now.Add(7 * 24 * time.Hour)
	var availableSlots []models.AvailableSlot

	// Loop over each day in the next 7 days.
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		dayStr := day.Format("2006-01-02")
		// Calculate the midnight of the current day.
		dayMidnight := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())

		// Iterate over each timeslot template configured for the provider.
		for _, ts := range provider.TimeSlots {
			// Compute absolute start/end times by adding minutes from midnight.
			absStart := dayMidnight.Add(time.Duration(ts.Start) * time.Minute)
			absEnd := dayMidnight.Add(time.Duration(ts.End) * time.Minute)

			// For today (dayOffset == 0): skip slots that have already ended.
			if dayOffset == 0 && absEnd.Before(now) {
				continue
			}
			// For any day: if the slot's start is after the overall cutoff, skip it.
			if absStart.After(cutoff) {
				continue
			}
			// For the final day in the window (dayOffset == 6): ensure the slot is complete within the window.
			if dayOffset == 6 && absEnd.After(cutoff) {
				continue
			}

			// Instantiate an AvailableSlot, attaching the date.
			var slot models.AvailableSlot
			slot.Start = ts.Start
			slot.End = ts.End
			slot.UnitType = ts.UnitType
			slot.Date = dayStr

			// Use denormalized aggregates (if present) instead of summing bookings in real time.
			switch ts.SlotModel {
			case "urgency":
				if ts.Urgency == nil {
					continue
				}
				// For urgency, standard capacity is reduced by reserved priority.
				normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority

				usageStandard := ts.BookedUnitsStandard // denormalized standard bookings
				usagePriority := ts.BookedUnitsPriority // denormalized priority bookings

				remainingStandard := normalCapacity - usageStandard
				remainingPriority := ts.Urgency.ReservedPriority - usagePriority

				slot.RegularCapacityRemaining = remainingStandard
				slot.PriorityCapacityRemaining = remainingPriority
				slot.RegularPricePerUnit = ts.Urgency.BasePrice
				slot.PriorityPricePerUnit = ts.Urgency.BasePrice * (1 + ts.Urgency.PrioritySurchargeRate)

				// If capacity is below 30%, add a notification message.
				if normalCapacity > 0 && float64(remainingStandard)/float64(normalCapacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining for standard bookings", remainingStandard, ts.UnitType)
				}
				if ts.Urgency.ReservedPriority > 0 && float64(remainingPriority)/float64(ts.Urgency.ReservedPriority) < 0.3 {
					if slot.Message != "" {
						slot.Message += " | "
					}
					slot.Message += fmt.Sprintf("Only %d %s remaining for priority bookings", remainingPriority, ts.UnitType)
				}

			case "earlybird":
				if ts.EarlyBird == nil {
					continue
				}
				// For earlybird, we assume one bucket; use denormalized field.
				usage := ts.BookedUnitsStandard
				remaining := ts.Capacity - usage
				nextPrice := GetEarlyBirdNextUnitPrice(*ts.EarlyBird, ts.Capacity, usage)
				slot.RegularCapacityRemaining = remaining
				slot.RegularPricePerUnit = nextPrice

				if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}

			default: // "flatrate" (or standard model).
				if ts.Flatrate == nil {
					continue
				}
				usage := ts.BookedUnitsStandard
				remaining := ts.Capacity - usage
				slot.RegularCapacityRemaining = remaining
				slot.RegularPricePerUnit = ts.Flatrate.BasePrice

				if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
					slot.Message = fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
				}
			}

			availableSlots = append(availableSlots, slot)
		}
	}

	// Optionally sort available slots by their start time (for client display).
	sort.Slice(availableSlots, func(i, j int) bool {
		return availableSlots[i].Start < availableSlots[j].Start
	})

	return availableSlots, nil
}

func (se *DefaultSchedulingEngine) BookSlot(provider models.Provider, date string, slot models.AvailableSlot, booking models.Booking) error {
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Validate that the booking's time window falls within the selected slot.
	if booking.Start < slot.Start || booking.End > slot.End {
		return fmt.Errorf("booking time [%d, %d] is not within slot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	// 2. Locate the matching TimeSlot template, ensuring that the date matches.
	var ts *models.TimeSlot
	for _, candidate := range provider.TimeSlots {
		// Assume candidate.Date is set when recurring slots are instantiated.
		if candidate.Start == slot.Start && candidate.End == slot.End && candidate.Date == date {
			ts = &candidate
			break
		}
	}
	if ts == nil {
		return fmt.Errorf("timeslot configuration not found for slot [%d-%d] on date %s", slot.Start, slot.End, date)
	}

	// 3. Calculate total price and perform capacity validation based on the slot model.
	var totalPrice float64
	switch ts.SlotModel {
	case "urgency":
		if ts.Urgency == nil {
			return fmt.Errorf("urgency slot data missing")
		}
		normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority
		if booking.Priority {
			usagePriority, err := se.Repo.SumOverlappingBookingsForPriority(provider.ID, date, ts.Start, ts.End)
			if err != nil {
				return fmt.Errorf("error checking priority usage: %w", err)
			}
			if usagePriority+booking.Units > ts.Urgency.ReservedPriority {
				return fmt.Errorf("priority booking exceeds reserved capacity; current usage %d, requested %d, reserved %d", usagePriority, booking.Units, ts.Urgency.ReservedPriority)
			}
			totalPrice = CalculateUrgencyPrice(*ts.Urgency, booking.Units, true)
		} else {
			usageStandard, err := se.Repo.SumOverlappingBookingsForStandard(provider.ID, date, ts.Start, ts.End)
			if err != nil {
				return fmt.Errorf("error checking standard usage: %w", err)
			}
			if usageStandard+booking.Units > normalCapacity {
				return fmt.Errorf("only priority capacity remains; standard booking cannot be accepted")
			}
			totalPrice = CalculateUrgencyPrice(*ts.Urgency, booking.Units, false)
		}

	case "earlybird":
		if ts.EarlyBird == nil {
			return fmt.Errorf("earlybird slot data missing")
		}
		usage, err := se.Repo.SumOverlappingBookings(provider.ID, date, ts.Start, ts.End)
		if err != nil {
			return fmt.Errorf("error checking total usage: %w", err)
		}
		if usage+booking.Units > ts.Capacity {
			return fmt.Errorf("booking exceeds slot capacity; current usage %d, requested %d, capacity %d", usage, booking.Units, ts.Capacity)
		}
		totalPrice = CalculateEarlyBirdPrice(*ts.EarlyBird, ts.Capacity, usage, booking.Units)

	default: // "flatrate" or standard.
		if ts.Flatrate == nil {
			return fmt.Errorf("flatrate slot data missing")
		}
		usage, err := se.Repo.SumOverlappingBookings(provider.ID, date, ts.Start, ts.End)
		if err != nil {
			return fmt.Errorf("error checking current usage: %w", err)
		}
		if usage+booking.Units > ts.Capacity {
			return fmt.Errorf("booking exceeds slot capacity; current usage %d, requested %d, capacity %d", usage, booking.Units, ts.Capacity)
		}
		totalPrice = CalculateFlatratePrice(*ts.Flatrate, booking.Units)
	}

	// 4. Finalize the booking record.
	booking.ProviderID = provider.ID
	booking.Date = date
	booking.CreatedAt = time.Now()
	booking.TotalPrice = totalPrice

	// 5. Persist the booking record to reserve the slot immediately.
	if err := se.Repo.CreateBooking(&booking); err != nil {
		return fmt.Errorf("error creating booking: %w", err)
	}

	// 6. Update the denormalized aggregates on the TimeSlot using optimistic concurrency.
	var updateErr error
	if ts.SlotModel == "urgency" {
		if booking.Priority {
			updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, *ts, date, booking.Units, true, ts.Version)
		} else {
			updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, *ts, date, booking.Units, false, ts.Version)
		}
	} else {
		updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, *ts, date, booking.Units, false, ts.Version)
	}
	if updateErr != nil {
		return fmt.Errorf("failed to update timeslot aggregates: %w", updateErr)
	}

	// 7. Block the slot if capacity is reached.
	var blockSlot bool
	switch ts.SlotModel {
	case "urgency":
		usageStandard, _ := se.Repo.SumOverlappingBookingsForStandard(provider.ID, date, ts.Start, ts.End)
		usagePriority, _ := se.Repo.SumOverlappingBookingsForPriority(provider.ID, date, ts.Start, ts.End)
		if usageStandard >= (ts.Capacity-ts.Urgency.ReservedPriority) && usagePriority >= ts.Urgency.ReservedPriority {
			blockSlot = true
		}
	default:
		updatedUsage, _ := se.Repo.SumOverlappingBookings(provider.ID, date, ts.Start, ts.End)
		if updatedUsage >= ts.Capacity {
			blockSlot = true
		}
	}
	if blockSlot {
		block := models.Blocked{
			ProviderID:  provider.ID,
			Date:        date,
			Start:       ts.Start,
			End:         ts.End,
			Reason:      "capacity reached",
			ServiceType: provider.ServiceType,
		}
		if err := se.Repo.CreateBlockedInterval(&block); err != nil {
			fmt.Printf("warning: failed to create block: %v\n", err)
		}
	}

	// 8. Payment follow-up:
	// Payment follow-up: For pre-payment providers, wait up to 5 minutes for payment confirmation.
	if provider.PrePaymentRequired {
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
				// Payment failed: rollback aggregates and cancel booking.
				_ = se.Repo.CancelBooking(booking.ID)
				rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, *ts, date, booking.Units, booking.Priority, ts.Version)
				if rollbackErr != nil {
					fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
				}
				return fmt.Errorf("payment not confirmed: invoice status %s", invoice.Status)
			}
			// Payment succeeded: continue normally.
		case err := <-errCh:
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, *ts, date, booking.Units, booking.Priority, ts.Version)
			if rollbackErr != nil {
				fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
			}
			return fmt.Errorf("payment processing error: %w", err)
		case <-time.After(5 * time.Minute):
			// Payment timeout: rollback aggregates and cancel booking.
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, *ts, date, booking.Units, booking.Priority, ts.Version)
			if rollbackErr != nil {
				fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
			}
			return fmt.Errorf("payment processing timed out; booking cancelled")
		}
	} else {
		// Cash-on-service: bypass payment processing.
		booking.PaymentMethod = "cash-on-service"
		booking.PaymentStatus = "pending"
	}

	return nil
}
