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
type Profile struct {
	ProviderName     string `bson:"provider_name" json:"provider_name,omitempty"`
	Email            string `bson:"email" json:"email,omitempty"`
	PhoneNumber      string `bson:"phone_number" json:"phone_number,omitempty"`
	Status           string `bson:"status" json:"status,omitempty"`
	AdvancedVerified bool   `bson:"advanced_verified" json:"advanced_verified,omitempty"`
	ProfileImage     string `bson:"profile_image,omitempty" json:"profile_image,omitempty"`
}

type Provider struct {
	ID           string  `bson:"id" json:"id,omitempty"`
	Profile      Profile `bson:"profile" json:"profile"`
	LegalName    string  `bson:"legal_name" json:"legal_name,omitempty"`
	Password     string  `bson:"-" json:"password,omitempty"`
	PasswordHash string  `bson:"password_hash" json:"-"`
	Token        string  `bson:"-" json:"token,omitempty"`
	TokenHash    string  `bson:"token_hash" json:"-"`
	ProviderType string  `bson:"provider_type" json:"provider_type,omitempty"`

	ServiceType string   `bson:"service_type" json:"service_type,omitempty"`
	Location    string   `bson:"location" json:"location,omitempty"`
	LocationGeo GeoPoint `bson:"location_geo" json:"location_geo"`

	KYPDocument         string `bson:"kyp_document" json:"kyp_document,omitempty"`
	VerificationStatus  string `bson:"verification_status" json:"verification_status,omitempty"`
	VerificationLevel   string `bson:"verification_level" json:"verification_level,omitempty"`
	KYPVerificationCode string `bson:"kyp_verification_code" json:"kyp_verification_code,omitempty"`

	InsuranceDocs []string `bson:"insurance_docs,omitempty" json:"insurance_docs,omitempty"`
	TaxPIN        string   `bson:"tax_pin,omitempty" json:"tax_pin,omitempty"`

	Rating            float64            `bson:"rating" json:"rating,omitempty"`
	CompletedBookings int                `bson:"completed_bookings" json:"completed_bookings,omitempty"`
	HistoricalRecords []HistoricalRecord `bson:"historical_records" json:"historical_records,omitempty"`
	TimeSlots         []TimeSlot         `bson:"time_slots" json:"time_slots,omitempty"`

	AcceptedPaymentMethods []string `bson:"accepted_payment_methods" json:"accepted_payment_methods,omitempty"`
	PrePaymentRequired     bool     `bson:"pre_payment_required" json:"pre_payment_required,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at,omitempty"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at,omitempty"`
}

type ProviderDTO struct {
	ID          string   `json:"id"`
	Profile     Profile  `json:"profile"`
	ServiceType string   `json:"service_type"`
	Location    string   `json:"location"`
	LocationGeo GeoPoint `json:"location_geo"`
	CreatedAt   string   `json:"created_at"`
	Preferred   bool     `json:"preferred"`
}
