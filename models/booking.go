// File: bloomify/models/booking.go
package models

import "time"

// Booking represents the stored booking record.
type Booking struct {
	ID            string    `bson:"id" json:"id"`
	ProviderID    string    `bson:"providerId" json:"providerId"`
	TimeSlotID    string    `bson:"timeSlotId" json:"timeSlotId"`
	UserID        string    `bson:"userId" json:"userId"`
	Units         int       `bson:"units" json:"units"`
	UnitType      string    `bson:"unitType" json:"unitType"`
	TotalPrice    float64   `bson:"totalPrice" json:"totalPrice"`
	Status        string    `bson:"status" json:"status"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
	Date          string    `bson:"date" json:"date"`
	Start         int       `bson:"start" json:"start"`
	End           int       `bson:"end" json:"end"`
	Priority      bool      `bson:"priority,omitempty" json:"priority,omitempty"`
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"`
	PaymentStatus string    `bson:"paymentStatus" json:"paymentStatus"`
}

// SubscriptionDetails represents the parameters the client sends for a subscription.
type SubscriptionDetails struct {
	StartDate time.Time `bson:"startDate" json:"startDate"` // When the recurring bookings should begin
	EndDate   time.Time `bson:"endDate" json:"endDate"`     // Last day for which bookings are to be generated
	PlanType  string    `bson:"planType" json:"planType"`   // e.g., "daily", "weekly", etc.
}

// BookingRequest is the struct sent by the client when requesting a booking.
type BookingRequest struct {
	ProviderID    string               `json:"providerId"`
	UserID        string               `json:"userId"`
	Date          string               `json:"date,omitempty"`
	Start         int                  `json:"start,omitempty"`
	End           int                  `json:"end,omitempty"`
	Units         int                  `json:"units,omitempty"`
	Priority      bool                 `json:"priority,omitempty"`
	Subscription  *SubscriptionDetails `json:"subscription,omitempty"`
	PaymentMethod string               `json:"paymentMethod,omitempty"`
	CustomOption  *CustomOption        `json:"customOption,omitempty"`
}

// SubscriptionBooking represents a subscription record maintained for a provider.
type SubscriptionBooking struct {
	ID                string            `bson:"id" json:"id"`
	ProviderID        string            `bson:"providerId" json:"providerId"`
	SubscriptionModel SubscriptionModel `bson:"subscriptionModel" json:"subscriptionModel"`
	SubscriberIDs     []string          `bson:"subscriberIds" json:"subscriberIds"`
	CreatedAt         time.Time         `bson:"createdAt" json:"createdAt"`
}

type SubscriptionModel struct {
	Plan       string   `bson:"plan" json:"plan"`             // e.g., "5-day weekly", "weekend-only"
	ActiveDays []string `bson:"activeDays" json:"activeDays"` // e.g., ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
	Discount   float64  `bson:"discount" json:"discount"`     // e.g., 0.9 for a 10% discount on subscription bookings
}

// BookingConfirmation represents the result of a successful booking validation.
type BookingConfirmation struct {
	BookingID  string  `bson:"bookingId" json:"bookingId"`
	TotalPrice float64 `bson:"totalPrice" json:"totalPrice"`
	Message    string  `bson:"message,omitempty" json:"message,omitempty"`
}

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

// Invoice represents an invoice generated after processing an in-app payment.
type Invoice struct {
	InvoiceID     string    `bson:"invoiceId" json:"invoiceId"`         // Unique invoice identifier.
	BookingID     string    `bson:"bookingId" json:"bookingId"`         // Associated booking ID.
	Amount        float64   `bson:"amount" json:"amount"`               // The amount charged.
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"` // e.g., "inApp"
	Status        string    `bson:"status" json:"status"`               // e.g., "paid"
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`         // Timestamp of invoice creation.
}
