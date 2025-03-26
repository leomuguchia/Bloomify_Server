// File: bloomify/models/booking.go
package models

import "time"

type Booking struct {
	ID            string    `bson:"id" json:"id"`                                 // Unique booking identifier (e.g., UUID)
	ProviderID    string    `bson:"providerId" json:"providerId"`                 // Provider who was booked
	ProviderName  string    `bson:"providerName" json:"providerName"`             // Provider who was booked
	UserID        string    `bson:"userId" json:"userId"`                         // User who made the booking
	Date          string    `bson:"date" json:"date"`                             // Booking date in "YYYY-MM-DD" format
	Units         int       `bson:"units" json:"units"`                           // Number of capacity units booked (e.g., 1, 2, etc.)
	UnitType      string    `bson:"unitType" json:"unitType"`                     // Measurement unit (e.g., "child", "kg", "hour")
	TotalPrice    float64   `bson:"totalPrice" json:"totalPrice"`                 // Calculated total price
	Status        string    `bson:"status" json:"status"`                         // e.g., "Confirmed", "Pending"
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`                   // Timestamp when booking was created
	Start         int       `bson:"start" json:"start"`                           // Booking start time (minutes from midnight)
	End           int       `bson:"end" json:"end"`                               // Booking end time (minutes from midnight)
	Priority      bool      `bson:"priority,omitempty" json:"priority,omitempty"` // Indicates if the booking is under the urgency (priority) bucket
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"`           // Payment method used (e.g., "inApp")
	PaymentStatus string    `bson:"paymentStatus" json:"paymentStatus"`           // e.g., "pending", "paid", "cancelled"
}

type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	UnitType     string `json:"unitType"`
	ProviderTerm string `json:"providerTerm"`
}

// BookingConfirmation represents the result of a successful booking validation.
type BookingConfirmation struct {
	BookingID  string  `bson:"bookingId" json:"bookingId"`
	TotalPrice float64 `bson:"totalPrice" json:"totalPrice"`
	Message    string  `bson:"message,omitempty" json:"message,omitempty"`
}
