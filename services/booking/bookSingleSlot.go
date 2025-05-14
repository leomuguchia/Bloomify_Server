package booking

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"bloomify/models"
	"bloomify/utils"
)

func (se *DefaultSchedulingEngine) bookSingleSlot(
	provider models.Provider,
	date string,
	slot models.TimeSlot,
	booking *models.Booking,
	customOption models.CustomOption,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[bookSingleSlot] Panic recovered: %v", r)
			err = fmt.Errorf("internal error occurred during booking: %v", r)
		}
	}()

	log.Printf("[bookSingleSlot] Start - Booking [%d, %d], Slot [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)

	// Step 1: Bounds Check
	if booking.Start < slot.Start || booking.End > slot.End {
		err := fmt.Errorf("booking time [%d, %d] outside slot bounds [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
		log.Printf("[bookSingleSlot] Bounds check failed: %v", err)
		return err
	}

	// Step 2: Validate & Price
	log.Printf("[bookSingleSlot] Validating and pricing booking for provider %s, slot %s", provider.ID, slot.ID)
	confirmation, err := ValidateAndBook(
		provider.ID, slot, *booking,
		provider.ServiceCatalogue, &customOption,
	)
	if err != nil {
		log.Printf("[bookSingleSlot] Validation failed: %v", err)
		return fmt.Errorf("validation failed: %w", err)
	}
	log.Printf("[bookSingleSlot] Validation succeeded. TotalPrice: %.2f", confirmation.TotalPrice)

	// Step 3: Prepare Booking Fields
	now := time.Now()
	booking.ID = uuid.New().String()
	booking.ProviderID = provider.ID
	booking.Date = date
	booking.CreatedAt = now
	booking.TotalPrice = confirmation.TotalPrice
	booking.TimeSlotID = slot.ID
	log.Printf("[bookSingleSlot] Booking populated: %+v", *booking)

	// Step 4: Process Payment
	payReq := models.PaymentRequest{
		UserID:   booking.UserID,
		Amount:   confirmation.TotalPrice,
		Currency: booking.UserPayment.Currency,
		Method:   booking.UserPayment.PaymentMethod,
	}
	log.Printf("[bookSingleSlot] Processing payment for user %s via %s", payReq.UserID, payReq.Method)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if se.PaymentHandler == nil {
		utils.Logger.Error("PaymentHandler is nil in scheduling engine!")
		return errors.New("internal server error: PaymentHandler not initialized")
	}

	invoice, payErr := se.PaymentHandler.ProcessPayment(ctx, payReq)
	if payErr != nil {
		log.Printf("[bookSingleSlot] Payment failed: %v", payErr)
		return fmt.Errorf("payment failed: %w", payErr)
	}
	log.Printf("[bookSingleSlot] Payment successful. Invoice ID: %s", invoice.InvoiceID)

	booking.Invoice = *invoice
	booking.Status = invoice.Status

	// Step 5: Insert Booking Transactionally
	log.Printf("[bookSingleSlot] Saving booking to DB")
	if err := se.Repo.BookSingleSlotTransactionally(ctx, provider.ID, date, slot, booking); err != nil {
		log.Printf("[bookSingleSlot] DB transaction failed: %v", err)
		return fmt.Errorf("booking transaction failed: %w", err)
	}
	log.Printf("[bookSingleSlot] Booking saved. Booking ID: %s", booking.ID)

	// Step 6: Capacity Re-check
	used, err := se.Repo.SumOverlappingBookings(provider.ID, date, slot.Start, slot.End)
	if err != nil {
		log.Printf("[bookSingleSlot] Capacity check error: %v", err)
		return fmt.Errorf("capacity check failed: %w", err)
	}
	log.Printf("[bookSingleSlot] Capacity usage: %d/%d", used, slot.Capacity)

	if used >= slot.Capacity {
		log.Printf("[bookSingleSlot] Capacity maxed. Blocking slot %s", slot.ID)
		if err := se.Repo.SetEmbeddedTimeSlotBlocked(provider.ID, slot.ID, date, true, "capacity reached"); err != nil {
			log.Printf("[bookSingleSlot] Failed to block slot %s: %v", slot.ID, err)
		}
	}

	// Step 7: Done
	log.Printf("[bookSingleSlot] Booking complete. ID: %s", booking.ID)
	return nil
}
