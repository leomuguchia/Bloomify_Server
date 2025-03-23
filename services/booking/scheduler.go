package booking

import (
	"context"
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	schedulerRepo "bloomify/database/repository/scheduler"
	"bloomify/models"
)

// DefaultSchedulingEngine is our production‑grade scheduler.
type DefaultSchedulingEngine struct {
	Repo           schedulerRepo.SchedulerRepository
	PaymentHandler PaymentProcessor
	ProviderRepo   providerRepo.ProviderRepository
}

// AvailableSlotsResult holds both the user‑facing slots and a mapping to full TimeSlot objects.
type AvailableSlotsResult struct {
	Slots   []models.AvailableSlot
	Mapping map[string]models.TimeSlot
}

// GetAvailableTimeSlots returns enriched available slots and a mapping from each slot's ID to its full TimeSlot.
func (se *DefaultSchedulingEngine) GetAvailableTimeSlots(provider models.Provider, weekIndex int) (AvailableSlotsResult, error) {
	// Retrieve enriched full TimeSlot objects (using ProviderRepo, etc.)
	enrichedSlots, err := GetEnrichedTimeslots(se.ProviderRepo, provider.ID)
	if err != nil {
		return AvailableSlotsResult{}, fmt.Errorf("failed to enrich timeslots: %w", err)
	}

	// Determine the booking window.
	var minDate, maxDate time.Time
	for i, ts := range enrichedSlots {
		d, err := time.Parse("2006-01-02", ts.Date)
		if err != nil {
			continue
		}
		if i == 0 || d.Before(minDate) {
			minDate = d
		}
		if i == 0 || d.After(maxDate) {
			maxDate = d
		}
	}
	if minDate.IsZero() || maxDate.IsZero() {
		return AvailableSlotsResult{}, fmt.Errorf("provider has no valid timeslot dates")
	}

	now := time.Now()
	startDate := now
	if now.Before(minDate) {
		startDate = minDate
	}
	weekZeroStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	weekStart := weekZeroStart.AddDate(0, 0, weekIndex*7)
	weekEnd := weekStart.AddDate(0, 0, 7)
	if weekEnd.After(maxDate.AddDate(0, 0, 1)) {
		weekEnd = maxDate.AddDate(0, 0, 1)
	}

	// Build available slots and mapping.
	slots, mapping, err := buildAvailableSlots(enrichedSlots, provider.ServiceCatalogue, weekStart, weekEnd, now)
	if err != nil {
		return AvailableSlotsResult{}, fmt.Errorf("failed to build available slots: %w", err)
	}

	return AvailableSlotsResult{
		Slots:   slots,
		Mapping: mapping,
	}, nil
}

// BookSlot validates and processes a booking for a given enriched timeslot.
func (se *DefaultSchedulingEngine) BookSlot(provider models.Provider, date string, slot models.TimeSlot, booking models.Booking) error {
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if booking.Start < slot.Start || booking.End > slot.End {
		return fmt.Errorf("booking time [%d, %d] is not within slot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	// Validate booking and calculate pricing.
	confirmation, err := ValidateAndBook(provider.ID, slot, booking, provider.ServiceCatalogue)
	if err != nil {
		return fmt.Errorf("provider booking validation failed: %w", err)
	}

	booking.ProviderID = provider.ID
	booking.Date = date
	booking.CreatedAt = time.Now()
	booking.TotalPrice = confirmation.TotalPrice

	if err := se.Repo.CreateBooking(&booking); err != nil {
		return fmt.Errorf("error creating booking: %w", err)
	}

	// Update timeslot aggregates.
	var updateErr error
	if slot.SlotModel == "urgency" {
		if booking.Priority {
			updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, slot, date, booking.Units, true, slot.Version)
		} else {
			updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, slot, date, booking.Units, false, slot.Version)
		}
	} else {
		updateErr = se.Repo.UpdateTimeSlotAggregates(provider.ID, slot, date, booking.Units, false, slot.Version)
	}
	if updateErr != nil {
		return fmt.Errorf("failed to update timeslot aggregates: %w", updateErr)
	}

	// Block the slot if capacity is reached.
	var blockSlot bool
	switch slot.SlotModel {
	case "urgency":
		usageStandard, _ := se.Repo.SumOverlappingBookingsForStandard(provider.ID, date, slot.Start, slot.End)
		usagePriority, _ := se.Repo.SumOverlappingBookingsForPriority(provider.ID, date, slot.Start, slot.End)
		if usageStandard >= (slot.Capacity-slot.Urgency.ReservedPriority) &&
			usagePriority >= slot.Urgency.ReservedPriority {
			blockSlot = true
		}
	default:
		updatedUsage, _ := se.Repo.SumOverlappingBookings(provider.ID, date, slot.Start, slot.End)
		if updatedUsage >= slot.Capacity {
			blockSlot = true
		}
	}
	if blockSlot {
		block := models.Blocked{
			ProviderID:  provider.ID,
			Date:        date,
			Start:       slot.Start,
			End:         slot.End,
			Reason:      "capacity reached",
			ServiceType: provider.ServiceCatalogue.ServiceType,
		}
		if err := se.Repo.CreateBlockedInterval(&block); err != nil {
			fmt.Printf("warning: failed to create block: %v\n", err)
		}
	}

	// Process payment for pre-payment providers.
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
				_ = se.Repo.CancelBooking(booking.ID)
				rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, slot, date, booking.Units, booking.Priority, slot.Version)
				if rollbackErr != nil {
					fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
				}
				return fmt.Errorf("payment not confirmed: invoice status %s", invoice.Status)
			}
		case err := <-errCh:
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, slot, date, booking.Units, booking.Priority, slot.Version)
			if rollbackErr != nil {
				fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
			}
			return fmt.Errorf("payment processing error: %w", err)
		case <-time.After(5 * time.Minute):
			_ = se.Repo.CancelBooking(booking.ID)
			rollbackErr := se.Repo.RollbackTimeSlotAggregates(provider.ID, slot, date, booking.Units, booking.Priority, slot.Version)
			if rollbackErr != nil {
				fmt.Printf("warning: failed to rollback aggregates for booking %s: %v\n", booking.ID, rollbackErr)
			}
			return fmt.Errorf("payment processing timed out; booking cancelled")
		}
	} else {
		booking.PaymentMethod = "cash-on-service"
		booking.PaymentStatus = "pending"
	}

	return nil
}
