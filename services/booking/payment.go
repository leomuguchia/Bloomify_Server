package booking

import (
	"errors"
	"fmt"
	"time"

	"bloomify/models"
)

// PaymentProcessor defines the interface for processing in‑app payments.
type PaymentProcessor interface {
	// ProcessPayment deducts the booking amount from the user's in‑app balance,
	// and returns an Invoice upon successful payment.
	ProcessPayment(booking *models.Booking) (*models.Invoice, error)
}

// InAppPaymentProcessor implements PaymentProcessor for in‑app payments.
type InAppPaymentProcessor struct{}

// ProcessPayment verifies the user's balance, deducts the booking amount,
// and returns an invoice. For this MVP, the balance check and deduction are simulated.
func (iap *InAppPaymentProcessor) ProcessPayment(booking *models.Booking) (*models.Invoice, error) {
	// Check the user's balance.
	balance, err := CheckUserBalance(booking.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}
	if balance < booking.TotalPrice {
		return nil, errors.New("insufficient balance for in-app payment")
	}

	// Deduct the booking amount from the user's balance.
	if err := DeductUserBalance(booking.UserID, booking.TotalPrice); err != nil {
		return nil, fmt.Errorf("failed to deduct balance: %w", err)
	}

	// Generate an invoice.
	invoice := &models.Invoice{
		InvoiceID:     generateInvoiceID(),
		BookingID:     booking.ID,
		Amount:        booking.TotalPrice,
		PaymentMethod: "inApp",
		Status:        "paid",
		CreatedAt:     time.Now(),
	}
	return invoice, nil
}

// CheckUserBalance simulates checking the user's balance.
// For MVP purposes, it returns a fixed amount.
func CheckUserBalance(userID string) (float64, error) {
	// Every user is assumed to have a balance of $1000.
	return 1000.0, nil
}

// DeductUserBalance simulates deducting an amount from the user's balance.
// For MVP purposes, we assume this operation always succeeds.
func DeductUserBalance(userID string, amount float64) error {
	// In a real implementation, this would update the user's balance atomically.
	return nil
}

// generateInvoiceID generates a unique invoice identifier.
func generateInvoiceID() string {
	return fmt.Sprintf("INV-%d", time.Now().UnixNano())
}
