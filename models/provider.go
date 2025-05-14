package models

import (
	"time"
)

type HistoricalRecord struct {
	RecordID         string           `bson:"recordId" json:"recordId"`                 // Unique identifier for the record.
	Date             time.Time        `bson:"date" json:"date"`                         // Date of the service.
	ServiceCatalogue ServiceCatalogue `bson:"serviceCatalogue" json:"serviceCatalogue"` // Full catalogue of services offered.
	TotalEarned      float64          `bson:"totalEarned" json:"totalEarned"`           // Earnings from this service.
	Review           *Review          `bson:"review,omitempty" json:"review,omitempty"` // Optional customer review.
	Bookings         []Booking        `bson:"bookings" json:"bookings"`                 // All bookings linked to this record.
}

type Review struct {
	Rating  float64 `bson:"rating" json:"rating"`   // Expected value between 1 and 5.
	Comment string  `bson:"comment" json:"comment"` // Customer's feedback.
}

// GeoPoint represents a GeoJSON Point.
type GeoPoint struct {
	Type        string    `bson:"type" json:"type"`               // Always "Point"
	Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [longitude, latitude]
}

type Profile struct {
	ProviderName     string   `bson:"providerName" json:"providerName,omitempty"`
	ProviderType     string   `bson:"providerType" json:"providerType,omitempty"`
	Email            string   `bson:"email" json:"email,omitempty"`
	PhoneNumber      string   `bson:"phoneNumber" json:"phoneNumber,omitempty"`
	Status           string   `bson:"status" json:"status,omitempty"`
	AdvancedVerified bool     `bson:"advancedVerified" json:"advancedVerified,omitempty"`
	ProfileImage     string   `bson:"profileImage" json:"profileImage,omitempty"`
	Address          string   `bson:"address" json:"address,omitempty"`
	Rating           float64  `bson:"rating" json:"rating,omitempty"`
	LocationGeo      GeoPoint `bson:"locationGeo" json:"locationGeo"`
}

type AdvancedVerification struct {
	InsuranceDocs []string `bson:"insuranceDocs,omitempty" json:"insuranceDocs,omitempty"`
	TaxPIN        string   `bson:"taxPin,omitempty" json:"taxPin,omitempty"`
}

type Security struct {
	Password     string `bson:"-" json:"password,omitempty"`
	PasswordHash string `bson:"passwordHash" json:"-"`
	Token        string `bson:"-" json:"token,omitempty"`
	TokenHash    string `bson:"tokenHash" json:"-"`
}

type BasicVerification struct {
	KYPDocument        string `bson:"kypDocument" json:"kypDocument,omitempty"`
	VerificationStatus string `bson:"verificationStatus" json:"verificationStatus,omitempty"`
	LegalName          string `bson:"legalName" json:"legalName,omitempty"`
	VerificationCode   string `bson:"verificationCode" json:"verificationCode,omitempty"`
}

type PaymentDetails struct {
	PaymentMethods  []string `bson:"paymentMethods" json:"paymentMethods"`   // e.g., ["cash", "card"]
	PreferredMethod string   `bson:"preferredMethod" json:"preferredMethod"` // e.g., "card"
	Currency        string   `bson:"currency" json:"currency"`               // e.g., "KES"

	// Stripe-related
	StripeAccountID string `bson:"stripeAccountID,omitempty" json:"stripeAccountID,omitempty"`
	StripeVerified  bool   `bson:"stripeVerified" json:"stripeVerified"`

	// Optional cash-only metadata
	AcceptsCash bool `bson:"acceptsCash" json:"acceptsCash"`

	// Timestamps
	LastUpdated time.Time `bson:"lastUpdated" json:"lastUpdated"`
}

type Provider struct {
	ID                   string                `bson:"id" json:"id,omitempty"`
	Profile              Profile               `bson:"profile" json:"profile"`
	Security             Security              `bson:"security" json:"security,omitzero"`
	ServiceCatalogue     ServiceCatalogue      `bson:"serviceCatalogue" json:"serviceCatalogue,omitzero"`
	VerificationLevel    string                `bson:"verificationLevel" json:"verificationLevel,omitempty"`
	BasicVerification    BasicVerification     `bson:"verification" json:"verification,omitzero"`
	AdvancedVerification AdvancedVerification  `bson:"advancedVerification" json:"advancedVerification,omitzero"`
	HistoricalRecords    []HistoricalRecord    `bson:"historicalRecords" json:"historicalRecords,omitempty"`
	TimeSlots            []TimeSlot            `bson:"timeSlots" json:"timeSlots,omitempty"`
	PaymentDetails       PaymentDetails        `bson:"paymentDetails" json:"paymentDetails,omitzero"`
	CompletedBookings    int                   `bson:"completedBookings" json:"completedBookings,omitempty"`
	CreatedAt            time.Time             `bson:"createdAt" json:"createdAt,omitzero"`
	UpdatedAt            time.Time             `bson:"updatedAt" json:"updatedAt,omitzero"`
	Devices              []Device              `bson:"devices,omitempty" json:"devices,omitempty"`
	SubscriptionEnabled  bool                  `bson:"subscriptionEnabled" json:"subscriptionEnabled"` // Set to true if provider qualifies for recurring bookings
	SubscriptionModel    SubscriptionModel     `bson:"subscriptionModel" json:"subscriptionModel"`     // NEW FIELD
	SubscriptionBooking  []SubscriptionBooking `bson:"subscriptionBooking,omitempty" json:"subscriptionBooking,omitempty"`
}
