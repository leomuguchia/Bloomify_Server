package models

import (
	"time"
)

// HistoricalRecord represents a record of a service provided by a provider.
type HistoricalRecord struct {
	RecordID         string    `bson:"recordId" json:"recordId"`
	Date             time.Time `bson:"date" json:"date"`
	Rating           float64   `bson:"rating" json:"rating"`
	ServiceProvided  string    `bson:"serviceProvided" json:"serviceProvided"`
	ServedWho        string    `bson:"servedWho" json:"servedWho"`
	TotalEarned      float64   `bson:"totalEarned" json:"totalEarned"`
	CustomerFeedback string    `bson:"customerFeedback" json:"customerFeedback"`
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

// ServiceCatalogue defines the offerings for a service provider.
type ServiceCatalogue struct {
	ServiceType   string             `bson:"serviceType" json:"serviceType,omitempty"`
	Mode          string             `bson:"mode" json:"mode,omitempty"` // e.g., "provider-to-user", "drop-off", "mobile-unit"
	CustomOptions map[string]float64 `bson:"customOptions" json:"customOptions,omitempty"`
}

// example:
// ServiceCatalogue{
// ServiceType: "cleaning",
// Mode: "provider-to-user",
// CustomOptions: map[string]float64{
// "standard": 1.0,
// "luxury":   1.2,
// "eco":      1.1,
// },
// }

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
	AcceptedPaymentMethods []string `bson:"acceptedPaymentMethods" json:"acceptedPaymentMethods,omitempty"`
	PrePaymentRequired     bool     `bson:"prePaymentRequired" json:"prePaymentRequired,omitempty"`
}

type Provider struct {
	ID                   string               `bson:"id" json:"id,omitempty"`
	Profile              Profile              `bson:"profile" json:"profile"`
	Security             Security             `bson:"security" json:"security,omitzero"`
	ServiceCatalogue     ServiceCatalogue     `bson:"serviceCatalogue" json:"serviceCatalogue,omitzero"`
	VerificationLevel    string               `bson:"verificationLevel" json:"verificationLevel,omitempty"`
	BasicVerification    BasicVerification    `bson:"verification" json:"verification,omitzero"`
	AdvancedVerification AdvancedVerification `bson:"advancedVerification" json:"advancedVerification,omitzero"`
	HistoricalRecords    []HistoricalRecord   `bson:"historicalRecords" json:"historicalRecords,omitempty"`
	TimeSlots            []TimeSlot           `bson:"timeSlots" json:"timeSlots,omitempty"`
	PaymentDetails       PaymentDetails       `bson:"paymentDetails" json:"paymentDetails,omitzero"`
	CompletedBookings    int                  `bson:"completedBookings" json:"completedBookings,omitempty"`
	CreatedAt            time.Time            `bson:"createdAt" json:"createdAt,omitzero"`
	UpdatedAt            time.Time            `bson:"updatedAt" json:"updatedAt,omitzero"`
	Devices              []Device             `bson:"devices,omitempty" json:"devices,omitempty"`
}

type ProviderDTO struct {
	ID               string           `json:"id"`
	Profile          Profile          `json:"profile"`
	ServiceCatalogue ServiceCatalogue `json:"serviceCatalogue"`
	LocationGeo      GeoPoint         `json:"locationGeo"`
	Preferred        bool             `json:"preferred"`
}

type ProviderAuthResponse struct {
	ID          string    `json:"id"`
	Token       string    `json:"token"`
	Profile     Profile   `json:"profile"`
	CreatedAt   time.Time `json:"created_at"`
	ServiceType string    `json:"service_type,omitempty"`
}
