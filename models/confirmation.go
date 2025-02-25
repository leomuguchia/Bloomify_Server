package models

import "time"

// BookingConfirmationResponse represents the final response returned after a booking is confirmed.
type BookingConfirmationResponse struct {
	BookingID     string    `bson:"booking_id" json:"booking_id"`                             // Unique booking identifier
	ProviderID    string    `bson:"provider_id" json:"provider_id"`                           // ID of the provider booked
	Date          string    `bson:"date" json:"date"`                                         // Date of the booking (e.g., "2025-02-25")
	Start         int       `bson:"start" json:"start"`                                       // Start time in minutes from midnight
	End           int       `bson:"end" json:"end"`                                           // End time in minutes from midnight
	PaymentMethod string    `bson:"payment_method" json:"payment_method"`                     // Payment method used (here, always "inApp")
	Confirmation  string    `bson:"confirmation" json:"confirmation"`                         // Confirmation message
	InvoiceID     string    `bson:"invoice_id,omitempty" json:"invoice_id,omitempty"`         // Invoice ID (if applicable)
	Amount        float64   `bson:"amount,omitempty" json:"amount,omitempty"`                 // Amount charged
	InvoiceStatus string    `bson:"invoice_status,omitempty" json:"invoice_status,omitempty"` // Status of the invoice (e.g., "paid")
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
}
