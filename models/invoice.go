package models

import "time"

// Invoice represents an invoice generated after processing an in-app payment.
type Invoice struct {
	InvoiceID     string    `bson:"invoice_id" json:"invoice_id"`         // Unique invoice identifier.
	BookingID     string    `bson:"booking_id" json:"booking_id"`         // Associated booking ID.
	Amount        float64   `bson:"amount" json:"amount"`                 // The amount charged.
	PaymentMethod string    `bson:"payment_method" json:"payment_method"` // e.g., "inApp"
	Status        string    `bson:"status" json:"status"`                 // e.g., "paid"
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`         // Timestamp of invoice creation.
}
