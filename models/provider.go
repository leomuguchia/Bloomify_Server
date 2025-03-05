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
	ID string `bson:"id" json:"id"`

	// Identification & Contact
	ProviderName string `bson:"provider_name" json:"provider_name"` // Public business/provider name
	LegalName    string `bson:"legal_name" json:"legal_name"`       // Legal name as per government ID
	Email        string `bson:"email" json:"email"`
	PhoneNumber  string `bson:"phone_number" json:"phone_number"`

	// Authentication
	Password     string `bson:"-" json:"password,omitempty"` // Transient field for registration
	PasswordHash string `bson:"password_hash" json:"-"`
	TokenHash    string `bson:"token_hash" json:"-"`

	// Service & Location
	ServiceType string  `bson:"service_type" json:"service_type"` // e.g., "Cleaning", "Laundry"
	Location    string  `bson:"location" json:"location"`         // Street address
	Latitude    float64 `bson:"latitude" json:"latitude"`         // Map coordinate
	Longitude   float64 `bson:"longitude" json:"longitude"`       // Map coordinate

	// Verification (KYP)
	KYPDocument         string `bson:"kyp_document" json:"kyp_document"`                   // URL or reference to government ID scan
	VerificationStatus  string `bson:"verification_status" json:"verification_status"`     // e.g., pending, verified, rejected
	VerificationLevel   string `bson:"verification_level" json:"verification_level"`       // e.g., basic, advanced
	KYPVerificationCode string `bson:"kyp_verification_code" json:"kyp_verification_code"` // Cryptographic code returned by external KYP service

	// Optional Advanced Verification
	InsuranceDocs []string `bson:"insurance_docs,omitempty" json:"insurance_docs,omitempty"` // Insurance and certification docs
	TaxPIN        string   `bson:"tax_pin,omitempty" json:"tax_pin,omitempty"`               // Business tax ID

	// Activity & Ratings
	Rating            float64            `bson:"rating" json:"rating"`                         // Average rating
	CompletedBookings int                `bson:"completed_bookings" json:"completed_bookings"` // Count of completed bookings
	HistoricalRecords []HistoricalRecord `bson:"historical_records" json:"historical_records"`
	TimeSlots         []TimeSlot         `bson:"time_slots" json:"time_slots"` // Pre-defined booking windows

	// Payment & Pre-Payment
	AcceptedPaymentMethods []string `bson:"accepted_payment_methods" json:"accepted_payment_methods"`
	PrePaymentRequired     bool     `bson:"pre_payment_required" json:"pre_payment_required"`

	// Metadata
	Verified  bool      `bson:"verified" json:"verified"` // Indicates if the provider is fully verified on the platform
	Status    string    `bson:"status" json:"status"`     // e.g., active, suspended, pending
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
