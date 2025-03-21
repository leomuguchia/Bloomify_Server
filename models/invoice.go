package models

import "time"

// Invoice represents an invoice generated after processing an in-app payment.
type Invoice struct {
	InvoiceID     string    `bson:"invoiceId" json:"invoiceId"`         // Unique invoice identifier.
	BookingID     string    `bson:"bookingId" json:"bookingId"`         // Associated booking ID.
	Amount        float64   `bson:"amount" json:"amount"`               // The amount charged.
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"` // e.g., "inApp"
	Status        string    `bson:"status" json:"status"`               // e.g., "paid"
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`         // Timestamp of invoice creation.
}
