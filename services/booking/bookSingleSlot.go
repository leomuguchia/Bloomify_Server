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
	customOption models.CustomOptionResponse,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[bookSingleSlot] Panic recovered: %v", r)
			err = fmt.Errorf("internal error occurred during booking: %v", r)
		}
	}()

	log.Printf("[bookSingleSlot] Start - Slot [%v], CustomOption [%v]", slot, customOption)

	if booking.Start < slot.Start || booking.End > slot.End {
		return fmt.Errorf("booking time [%d, %d] outside slot bounds [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
	}

	log.Printf("[bookSingleSlot] Validating and pricing booking for provider %s, slot %s", provider.ID, slot.ID)
	confirmation, err := ValidateAndBook(provider.ID, slot, *booking, &customOption)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	log.Printf("[bookSingleSlot] Validation succeeded. TotalPrice: %.2f", confirmation.TotalPrice)

	now := time.Now()
	booking.ID = uuid.New().String()
	booking.ProviderID = provider.ID
	booking.Date = date
	booking.CreatedAt = now
	booking.TotalPrice = confirmation.TotalPrice
	booking.TimeSlotID = slot.ID

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if se.PaymentHandler == nil {
		utils.Logger.Error("PaymentHandler is nil in scheduling engine!")
		return errors.New("internal server error: PaymentHandler not initialized")
	}

	invoice := &models.Invoice{
		InvoiceID: uuid.New().String(),
		UserID:    booking.UserID,
		Amount:    confirmation.TotalPrice,
		Currency:  booking.UserPayment.Currency,
		Method:    booking.UserPayment.PaymentMethod,
		Status:    "requires_capture",
		PaymentID: booking.UserPayment.PaymentIntentId,
		CreatedAt: now,
	}

	if invoice.Method == "cash" {
		payReq := models.PaymentRequest{
			UserID:   booking.UserID,
			Amount:   confirmation.TotalPrice,
			Currency: booking.UserPayment.Currency,
			Method:   "cash",
			Action:   "record",
			Metadata: map[string]string{
				"bookingId":  booking.ID,
				"providerId": provider.ID,
				"slotId":     slot.ID,
			},
		}
		invoice, err = se.PaymentHandler.ProcessPayment(ctx, payReq)
		if err != nil {
			return fmt.Errorf("cash payment failed: %w", err)
		}
	}

	booking.Invoice = *invoice
	booking.Status = invoice.Status

	if err := se.Repo.BookSingleSlotTransactionally(ctx, provider.ID, date, slot, booking); err != nil {
		if invoice.Method == "card" && invoice.PaymentID != "" {
			cancelReq := models.PaymentRequest{
				Method:          "card",
				PaymentIntentID: invoice.PaymentID,
				Action:          "cancel",
			}
			_, _ = se.PaymentHandler.ProcessPayment(ctx, cancelReq)
		}
		return fmt.Errorf("booking transaction failed: %w", err)
	}

	var paymentCaptureFailed bool
	if invoice.Method == "card" {
		captureReq := models.PaymentRequest{
			UserID:          booking.UserID,
			Method:          "card",
			PaymentIntentID: invoice.PaymentID,
			Action:          "capture",
			Amount:          invoice.Amount,
		}
		_, err = se.PaymentHandler.ProcessPayment(ctx, captureReq)
		if err != nil {
			paymentCaptureFailed = true
			booking.Status = "payment_required"
			cancelReq := models.PaymentRequest{
				UserID:          booking.UserID,
				Method:          "card",
				PaymentIntentID: invoice.PaymentID,
				Action:          "cancel",
			}
			_, _ = se.PaymentHandler.ProcessPayment(ctx, cancelReq)
		} else {
			booking.Status = "confirmed"
		}
	}

	// Handle user notifications
	if ok := se.NotifyUserWithBookingStatus(provider, booking, paymentCaptureFailed); !ok {
		log.Printf("[bookSingleSlot] Failed to notify user with booking status")
	}

	// Check capacity usage
	var used int
	if slot.CapacityMode == models.CapacitySingleUse {
		if err := se.TimeslotsRepo.SetTimeSlotBlockReason(ctx, provider.ID, slot.ID, date, true, "booked exclusively"); err != nil {
			log.Printf("[bookSingleSlot] Failed to block slot: %v", err)
		}
	} else {
		used, err = se.Repo.SumOverlappingBookings(provider.ID, date, slot.Start, slot.End, &booking.Priority)
		if err != nil {
			log.Printf("[bookSingleSlot] Capacity check error: %v", err)
		} else {
			log.Printf("[bookSingleSlot] Capacity usage: %d/%d", used, slot.Capacity)
			if used >= slot.Capacity {
				if err := se.TimeslotsRepo.SetTimeSlotBlockReason(ctx, provider.ID, slot.ID, date, true, "capacity full"); err != nil {
					log.Printf("[bookSingleSlot] Failed to block slot: %v", err)
				}
			}
		}
	}

	// Notify provider
	if ok := se.UpdateProviderWithBookingNotification(&provider, booking, slot, used); !ok {
		log.Printf("[bookSingleSlot] Failed to update provider with booking notification")
	}

	log.Printf("[bookSingleSlot] Booking complete. ID: %s", booking.ID)
	return nil
}
