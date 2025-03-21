// File: bloomify/models/booking_confirmation_response.go
package models

import "time"

// BookingConfirmationResponse represents the final response returned after a booking is confirmed.
type BookingConfirmationResponse struct {
	BookingID     string    `bson:"bookingId" json:"bookingId"`                             // Unique booking identifier
	ProviderID    string    `bson:"providerId" json:"providerId"`                           // ID of the provider booked
	Date          string    `bson:"date" json:"date"`                                       // Date of the booking (e.g., "2025-02-25")
	Start         int       `bson:"start" json:"start"`                                     // Start time in minutes from midnight
	End           int       `bson:"end" json:"end"`                                         // End time in minutes from midnight
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"`                     // Payment method used (here, always "inApp")
	Confirmation  string    `bson:"confirmation" json:"confirmation"`                       // Confirmation message
	InvoiceID     string    `bson:"invoiceId,omitempty" json:"invoiceId,omitempty"`         // Invoice ID (if applicable)
	Amount        float64   `bson:"amount,omitempty" json:"amount,omitempty"`               // Amount charged
	InvoiceStatus string    `bson:"invoiceStatus,omitempty" json:"invoiceStatus,omitempty"` // Status of the invoice (e.g., "paid")
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
}
