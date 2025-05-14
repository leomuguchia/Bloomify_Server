// File: bloomify/models/booking.go
package models

import "time"

// Booking represents the stored booking record.
type Booking struct {
	ID           string       `bson:"id" json:"id"`
	ProviderID   string       `bson:"providerId" json:"providerId"`
	TimeSlotID   string       `bson:"timeSlotId" json:"timeSlotId"`
	UserID       string       `bson:"userId" json:"userId"`
	Units        int          `bson:"units" json:"units"`
	UnitType     string       `bson:"unitType" json:"unitType"`
	TotalPrice   float64      `bson:"totalPrice" json:"totalPrice"`
	Status       string       `bson:"status" json:"status"`
	CreatedAt    time.Time    `bson:"createdAt" json:"createdAt"`
	Date         string       `bson:"date" json:"date"`
	Start        int          `bson:"start" json:"start"`
	End          int          `bson:"end" json:"end"`
	Priority     bool         `bson:"priority,omitempty" json:"priority,omitempty"`
	CustomOption CustomOption `bson:"customOption,omitempty" json:"customOption,omitzero"`
	Invoice      Invoice      `bson:"invoice,omitempty" json:"invoice,omitzero"`
	UserPayment  UserPayment  `bson:"userPayment" json:"userPayment,omitzero"`
}

type SubscriptionDetails struct {
	StartDate    time.Time `json:"startDate"`
	EndDate      time.Time `json:"endDate"`
	PlanType     string    `json:"planType"`               // "daily","weekly","monthly"
	ExemptedDays []string  `json:"exemptedDays,omitempty"` // for daily only
	Weekday      string    `json:"weekday,omitempty"`      // for weekly only, e.g. "Tuesday"
	DayOfMonth   int       `json:"dayOfMonth,omitempty"`   // for monthly only, e.g. 15
}

// BookingRequest is the struct sent by the client when requesting a booking.
type BookingRequest struct {
	ProviderID          string              `json:"providerId"`
	UserID              string              `json:"userId"`
	Date                string              `json:"date,omitempty"`
	Start               int                 `json:"start,omitempty"`
	End                 int                 `json:"end,omitempty"`
	Units               int                 `json:"units,omitempty"`
	UnitType            string              `json:"unitType,omitempty"`
	Priority            bool                `json:"priority,omitempty"`
	Subscription        bool                `json:"subscription"`
	SubscriptionDetails SubscriptionDetails `json:"subscriptionDetails,omitzero"`
	CustomOption        CustomOption        `json:"customOption,omitzero"`
	UserPayment         UserPayment         `json:"userPayment"`
}

type SubscriptionModel struct {
	Plan     string  `bson:"plan" json:"plan"`         // e.g., "5-day weekly", "weekend-only"
	Discount float64 `bson:"discount" json:"discount"` // e.g., 0.9 for a 10% discount on subscription bookings
}

// BookingConfirmation represents the result of a successful booking validation.
type BookingConfirmation struct {
	BookingID  string  `bson:"bookingId" json:"bookingId"`
	TotalPrice float64 `bson:"totalPrice" json:"totalPrice"`
	Message    string  `bson:"message,omitempty" json:"message,omitempty"`
}

type SubscriptionBooking struct {
	SuccessfulBookings []Booking `json:"successfulBookings"`
	Errors             []error   `json:"-"`
	ErrorCount         int       `json:"errorCount"`
}
