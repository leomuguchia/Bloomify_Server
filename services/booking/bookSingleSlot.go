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
		err := fmt.Errorf("booking time [%d, %d] outside slot bounds [%d, %d]", booking.Start, booking.End, slot.Start, slot.End)
		log.Printf("[bookSingleSlot] Bounds check failed: %v", err)
		return err
	}

	log.Printf("[bookSingleSlot] Validating and pricing booking for provider %s, slot %s", provider.ID, slot.ID)
	confirmation, err := ValidateAndBook(provider.ID, slot, *booking, &customOption)
	if err != nil {
		log.Printf("[bookSingleSlot] Validation failed: %v", err)
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
	log.Printf("[bookSingleSlot] Booking populated: %+v", *booking)

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
		CreatedAt: time.Now(),
	}

	if invoice.Method == "cash" {
		log.Printf("[bookSingleSlot] Recording cash payment")
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
			log.Printf("[bookSingleSlot] Cash payment failed: %v", err)
			return fmt.Errorf("cash payment failed: %w", err)
		}
	}

	booking.Invoice = *invoice
	booking.Status = invoice.Status

	log.Printf("[bookSingleSlot] Saving booking to DB")
	if err := se.Repo.BookSingleSlotTransactionally(ctx, provider.ID, date, slot, booking); err != nil {
		log.Printf("[bookSingleSlot] DB transaction failed: %v", err)

		// Cancel PaymentIntent if it was a card payment
		if invoice.Method == "card" && invoice.PaymentID != "" {
			log.Printf("[bookSingleSlot] Canceling payment intent due to DB failure")
			cancelReq := models.PaymentRequest{
				Method:          "card",
				PaymentIntentID: invoice.PaymentID,
				Action:          "cancel",
			}
			_, cancelErr := se.PaymentHandler.ProcessPayment(ctx, cancelReq)
			if cancelErr != nil {
				log.Printf("[bookSingleSlot] Failed to cancel payment intent: %v", cancelErr)
			} else {
				log.Printf("[bookSingleSlot] Payment intent canceled successfully")
			}
		}
		return fmt.Errorf("booking transaction failed: %w", err)
	}

	log.Printf("[bookSingleSlot] Booking saved. Booking ID: %s", booking.ID)

	var paymentCaptureFailed bool

	if invoice.Method == "card" {
		log.Printf("[bookSingleSlot] Capturing card payment")
		captureReq := models.PaymentRequest{
			UserID:          booking.UserID,
			Method:          "card",
			PaymentIntentID: invoice.PaymentID,
			Action:          "capture",
			Amount:          invoice.Amount,
		}
		_, err = se.PaymentHandler.ProcessPayment(ctx, captureReq)
		if err != nil {
			log.Printf("[bookSingleSlot] Payment capture failed: %v", err)
			paymentCaptureFailed = true
			booking.Status = "payment_failed"
			cancelReq := models.PaymentRequest{
				UserID:          booking.UserID,
				Method:          "card",
				PaymentIntentID: invoice.PaymentID,
				Action:          "cancel",
			}
			_, err = se.PaymentHandler.ProcessPayment(ctx, cancelReq)
		} else {
			log.Printf("[bookSingleSlot] Card payment captured")
		}
	}

	// Update user with booking info and send notification
	user, userErr := se.UserService.GetUserByID(booking.UserID)
	if userErr != nil {
		log.Printf("[bookSingleSlot] Failed to fetch user for notification: %v", userErr)
	} else {
		user.ActiveBookings = append(user.ActiveBookings, booking.ID)

		status := "paid"
		message := fmt.Sprintf("Booking ID %s confirmed. Payment captured successfully.", booking.ID)
		if paymentCaptureFailed {
			status = "payment_failed"
			message = fmt.Sprintf("Booking ID %s was created but payment failed. Please retry payment", booking.ID)
		}

		notification := models.Notification{
			ID:      uuid.New().String(),
			Type:    "booking_update",
			Message: message,
			Data: map[string]any{
				"bookingId": booking.ID,
				"status":    status,
				"amount":    booking.TotalPrice,
			},
			CreatedAt: time.Now(),
			Read:      false,
		}
		user.Notifications = append(user.Notifications, notification)
		user.UpdatedAt = time.Now()

		if _, updateErr := se.UserService.UpdateUser(*user); updateErr != nil {
			log.Printf("[bookSingleSlot] Failed to update user with booking info: %v", updateErr)
		}
	}

	// Post-booking slot capacity update
	used, err := se.Repo.SumOverlappingBookings(provider.ID, date, slot.Start, slot.End, &booking.Priority)
	if err != nil {
		log.Printf("[bookSingleSlot] Capacity check error: %v", err)
		return fmt.Errorf("capacity check failed: %w", err)
	}
	log.Printf("[bookSingleSlot] Capacity usage: %d/%d", used, slot.Capacity)

	if used >= slot.Capacity {
		log.Printf("[bookSingleSlot] Capacity maxed. Blocking slot %s", slot.ID)
		if err := se.TimeslotsRepo.SetTimeSlotBlockReason(ctx, provider.ID, slot.ID, date, true, "capacity reached"); err != nil {
			log.Printf("[bookSingleSlot] Failed to set block reason for slot %s: %v", slot.ID, err)
		}
	}

	log.Printf("[bookSingleSlot] Booking complete. ID: %s", booking.ID)
	return nil
}
