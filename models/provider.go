package models

import (
	"time"
)

// HistoricalRecord represents a record of a service provided by a provider.
type HistoricalRecord struct {
	RecordID         string    `bson:"record_id" json:"record_id"`                 // Unique identifier for the record
	Date             time.Time `bson:"date" json:"date"`                           // When the service was provided
	Rating           float64   `bson:"rating" json:"rating"`                       // Service rating
	ServiceProvided  string    `bson:"service_provided" json:"service_provided"`   // e.g., "Laundry", "Cleaning", "Chauffeur"
	ServedWho        string    `bson:"served_who" json:"served_who"`               // Information about the client served
	TotalEarned      float64   `bson:"total_earned" json:"total_earned"`           // Total earnings from the service
	CustomerFeedback string    `bson:"customer_feedback" json:"customer_feedback"` // Customer feedback
}

// Provider represents a service provider.
type Provider struct {
	ID                string             `bson:"id" json:"id"`
	Name              string             `bson:"name" json:"name"`
	Email             string             `bson:"email" json:"email"`
	Password          string             `bson:"-" json:"password,omitempty"` // Transient field; not persisted.
	PasswordHash      string             `bson:"password_hash" json:"-"`
	PhoneNumber       string             `bson:"phone_number" json:"phone_number"`
	TimeSlots         []TimeSlot         `bson:"time_slots" json:"time_slots"`     // Pre-defined booking windows
	ServiceType       string             `bson:"service_type" json:"service_type"` // e.g., "Cleaning", "Laundry", etc.
	Location          string             `bson:"location" json:"location"`         // Human-readable location
	Latitude          float64            `bson:"latitude" json:"latitude"`
	Longitude         float64            `bson:"longitude" json:"longitude"`
	Rating            float64            `bson:"rating" json:"rating"`                         // Average rating (0.0 to 5.0)
	CompletedBookings int                `bson:"completed_bookings" json:"completed_bookings"` // Count of completed bookings
	Verified          bool               `bson:"verified" json:"verified"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
	Status            string             `bson:"status" json:"status"`
	HistoricalRecords []HistoricalRecord `bson:"historical_records" json:"historical_records"`

	// PaymentOptions defines which payment methods the provider accepts.
	// For example, a provider may allow ["cash"] for post-payment or ["stripe"] for pre-payment.
	AcceptedPaymentMethods []string `bson:"accepted_payment_methods" json:"accepted_payment_methods"`

	// Derived field: if true, the provider requires pre-payment (e.g., via stripe or card)
	PrePaymentRequired bool `bson:"pre_payment_required" json:"pre_payment_required"`
}
