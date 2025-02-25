package booking

import (
	"fmt"
	"time"

	schedulerRepo "bloomify/database/repository/scheduler"
	"bloomify/models"
)

// BookingConfirmation defines methods to finalize a booking given a confirmation session.
type BookingConfirmation interface {
	// ConfirmSession performs all confirmation logic using the provided confirmation session.
	ConfirmSession(session models.BookingConfirmationSession) (*models.BookingConfirmationResponse, error)
}

// DefaultBookingConfirmation implements BookingConfirmation.
type DefaultBookingConfirmation struct {
	Repo           schedulerRepo.SchedulerRepository // For persisting bookings.
	PaymentHandler PaymentProcessor                  // In-app payment processor.
}

// ConfirmSession verifies the confirmation session details, processes payment concurrently,
// persists the booking, and returns a confirmation response.
func (bc *DefaultBookingConfirmation) ConfirmSession(session models.BookingConfirmationSession) (*models.BookingConfirmationResponse, error) {
	// 1. Verify essential session details.
	if session.SelectedProvider == "" {
		return nil, fmt.Errorf("no provider selected in confirmation session")
	}
	if len(session.Availability) == 0 {
		return nil, fmt.Errorf("no available interval found in confirmation session")
	}
	if session.PaymentMethod != "inApp" {
		return nil, fmt.Errorf("in-app payment is required")
	}

	// 2. Select the first available interval.
	selectedInterval := session.Availability[0]

	// 3. Re-check capacity using denormalized values.
	var capacityAvailable int
	if session.ServicePlan.Priority {
		capacityAvailable = selectedInterval.PriorityCapacityRemaining
	} else {
		capacityAvailable = selectedInterval.RegularCapacityRemaining
	}
	if session.ServicePlan.Units > capacityAvailable {
		return nil, fmt.Errorf("verification failed: requested units (%d) exceed available capacity (%d)",
			session.ServicePlan.Units, capacityAvailable)
	}

	// 4. Verify that the timeslot hasn't expired.
	bookingDate, err := time.Parse("2006-01-02", session.ServicePlan.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid booking date %q: %w", session.ServicePlan.Date, err)
	}
	dayMidnight := time.Date(bookingDate.Year(), bookingDate.Month(), bookingDate.Day(), 0, 0, 0, 0, time.Local)
	absEnd := dayMidnight.Add(time.Duration(selectedInterval.End) * time.Minute)
	if time.Now().After(absEnd) {
		return nil, fmt.Errorf("verification failed: the selected timeslot has expired")
	}

	// 5. Recalculate total price.
	var totalPrice float64
	if session.ServicePlan.Priority {
		totalPrice = selectedInterval.PriorityPricePerUnit * float64(session.ServicePlan.Units)
	} else {
		totalPrice = selectedInterval.RegularPricePerUnit * float64(session.ServicePlan.Units)
	}

	// 6. Construct the final booking record.
	finalBooking := models.Booking{
		ProviderID:    session.SelectedProvider,
		UserID:        session.UserID,
		Date:          session.ServicePlan.Date,
		Start:         selectedInterval.Start,
		End:           selectedInterval.End,
		Units:         session.ServicePlan.Units,
		PaymentMethod: session.PaymentMethod,
		TotalPrice:    totalPrice,
		CreatedAt:     time.Now(),
	}

	// 7. Process in-app payment concurrently.
	paymentCh := make(chan bool)
	errCh := make(chan error)
	go func() {
		_, payErr := bc.PaymentHandler.ProcessPayment(&finalBooking)
		if payErr != nil {
			errCh <- payErr
			return
		}
		paymentCh <- true
	}()

	select {
	case confirmed := <-paymentCh:
		if !confirmed {
			return nil, fmt.Errorf("payment processing failed")
		}
	case err := <-errCh:
		return nil, fmt.Errorf("payment processing error: %w", err)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("payment processing timed out")
	}

	// 8. Persist the final booking.
	if err := bc.Repo.CreateBooking(&finalBooking); err != nil {
		return nil, fmt.Errorf("failed to create final booking: %w", err)
	}

	// 9. Build and return the confirmation response.
	response := models.BookingConfirmationResponse{
		BookingID:     finalBooking.ID,
		ProviderID:    finalBooking.ProviderID,
		Date:          finalBooking.Date,
		Start:         finalBooking.Start,
		End:           finalBooking.End,
		PaymentMethod: finalBooking.PaymentMethod,
		Confirmation:  "Booking confirmed",
		CreatedAt:     finalBooking.CreatedAt,
		// Optionally include invoice details.
	}
	return &response, nil
}
