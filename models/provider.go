package models

import (
	"time"
)

// HistoricalRecord represents a record of a service provided by a provider.
type HistoricalRecord struct {
	RecordID         string    `bson:"record_id" json:"record_id"`
	Date             time.Time `bson:"date" json:"date"`
	Rating           float64   `bson:"rating" json:"rating"`
	ServiceProvided  string    `bson:"service_provided" json:"service_provided"`
	ServedWho        string    `bson:"served_who" json:"served_who"`
	TotalEarned      float64   `bson:"total_earned" json:"total_earned"`
	CustomerFeedback string    `bson:"customer_feedback" json:"customer_feedback"`
}

// GeoPoint represents a GeoJSON Point.
type GeoPoint struct {
	Type        string    `bson:"type" json:"type"`               // Always "Point"
	Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [longitude, latitude]
}

// Provider represents a service provider in the system.
type Provider struct {
	ID string `bson:"id" json:"id,omitempty"`

	// Identification & Contact
	ProviderName string `bson:"provider_name" json:"provider_name,omitempty"`
	LegalName    string `bson:"legal_name" json:"legal_name,omitempty"`
	Email        string `bson:"email" json:"email,omitempty"`
	PhoneNumber  string `bson:"phone_number" json:"phone_number,omitempty"`

	// Authentication
	Password     string `bson:"-" json:"password,omitempty"`
	PasswordHash string `bson:"password_hash" json:"-"`
	Token        string `bson:"-" json:"token,omitempty"`
	TokenHash    string `bson:"token_hash" json:"-"`

	// Service & Location
	ServiceType string   `bson:"service_type" json:"service_type,omitempty"`
	Location    string   `bson:"location" json:"location,omitempty"`
	LocationGeo GeoPoint `bson:"location_geo" json:"location_geo"` // GeoJSON representation

	// Verification (KYP)
	KYPDocument         string `bson:"kyp_document" json:"kyp_document,omitempty"`
	VerificationStatus  string `bson:"verification_status" json:"verification_status,omitempty"`
	VerificationLevel   string `bson:"verification_level" json:"verification_level,omitempty"`
	KYPVerificationCode string `bson:"kyp_verification_code" json:"kyp_verification_code,omitempty"`

	// Optional Advanced Verification
	InsuranceDocs []string `bson:"insurance_docs,omitempty" json:"insurance_docs,omitempty"`
	TaxPIN        string   `bson:"tax_pin,omitempty" json:"tax_pin,omitempty"`

	// Activity & Ratings
	Rating            float64            `bson:"rating" json:"rating,omitempty"`
	CompletedBookings int                `bson:"completed_bookings" json:"completed_bookings,omitempty"`
	HistoricalRecords []HistoricalRecord `bson:"historical_records" json:"historical_records,omitempty"`
	TimeSlots         []TimeSlot         `bson:"time_slots" json:"time_slots,omitempty"`

	// Payment & Pre-Payment
	AcceptedPaymentMethods []string `bson:"accepted_payment_methods" json:"accepted_payment_methods,omitempty"`
	PrePaymentRequired     bool     `bson:"pre_payment_required" json:"pre_payment_required,omitempty"`

	// Metadata
	AdvancedVerified bool      `bson:"verified" json:"advanced_verified,omitempty"`
	Status           string    `bson:"status" json:"status,omitempty"`
	CreatedAt        time.Time `bson:"created_at" json:"created_at,omitempty"`
	UpdatedAt        time.Time `bson:"updated_at" json:"updated_at,omitempty"`
}
