// camel case naming.
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
	ProviderName     string `bson:"providerName" json:"providerName,omitempty"`
	Email            string `bson:"email" json:"email,omitempty"`
	PhoneNumber      string `bson:"phoneNumber" json:"phoneNumber,omitempty"`
	Status           string `bson:"status" json:"status,omitempty"`
	AdvancedVerified bool   `bson:"advancedVerified" json:"advancedVerified,omitempty"`
	ProfileImage     string `bson:"profileImage" json:"profileImage,omitempty"`
	Address          string `bson:"address" json:"address,omitempty"`
}

type ServiceCatalogue struct {
	ServiceType   string                 `bson:"serviceType" json:"serviceType,omitempty"`
	Mode          string                 `bson:"mode" json:"mode,omitempty"`
	CustomOptions map[string]interface{} `bson:"customOptions" json:"customOptions,omitempty"`
}

type Provider struct {
	ID                     string             `bson:"id" json:"id,omitempty"`
	Profile                Profile            `bson:"profile" json:"profile"`
	LegalName              string             `bson:"legalName" json:"legalName,omitempty"`
	Password               string             `bson:"-" json:"password,omitempty"`
	PasswordHash           string             `bson:"passwordHash" json:"-"`
	Token                  string             `bson:"-" json:"token,omitempty"`
	TokenHash              string             `bson:"tokenHash" json:"-"`
	ProviderType           string             `bson:"providerType" json:"providerType,omitempty"`
	ServiceCatalogue       ServiceCatalogue   `bson:"serviceCatalogue" json:"serviceCatalogue,omitempty"`
	Location               string             `bson:"location" json:"location,omitempty"`
	LocationGeo            GeoPoint           `bson:"locationGeo" json:"locationGeo"`
	KYPDocument            string             `bson:"kypDocument" json:"kypDocument,omitempty"`
	VerificationStatus     string             `bson:"verificationStatus" json:"verificationStatus,omitempty"`
	VerificationLevel      string             `bson:"verificationLevel" json:"verificationLevel,omitempty"`
	KYPVerificationCode    string             `bson:"kypVerificationCode" json:"kypVerificationCode,omitempty"`
	InsuranceDocs          []string           `bson:"insuranceDocs,omitempty" json:"insuranceDocs,omitempty"`
	TaxPIN                 string             `bson:"taxPin,omitempty" json:"taxPin,omitempty"`
	Rating                 float64            `bson:"rating" json:"rating,omitempty"`
	CompletedBookings      int                `bson:"completedBookings" json:"completedBookings,omitempty"`
	HistoricalRecords      []HistoricalRecord `bson:"historicalRecords" json:"historicalRecords,omitempty"`
	TimeSlots              []TimeSlot         `bson:"timeSlots" json:"timeSlots,omitempty"`
	AcceptedPaymentMethods []string           `bson:"acceptedPaymentMethods" json:"acceptedPaymentMethods,omitempty"`
	PrePaymentRequired     bool               `bson:"prePaymentRequired" json:"prePaymentRequired,omitempty"`
	CreatedAt              time.Time          `bson:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt              time.Time          `bson:"updatedAt" json:"updatedAt,omitempty"`
	Devices                []Device           `bson:"devices,omitempty" json:"devices,omitempty"`
}

type ProviderDTO struct {
	ID               string           `json:"id"`
	Profile          Profile          `json:"profile"`
	ServiceCatalogue ServiceCatalogue `json:"serviceCatalogue"`
	LocationGeo      GeoPoint         `json:"locationGeo"`
	Preferred        bool             `json:"preferred"`
}

type ProviderAuthResponse struct {
	ID           string    `json:"id"`
	Token        string    `json:"token"`
	Profile      Profile   `json:"profile"`
	CreatedAt    time.Time `json:"created_at"`
	ProviderType string    `json:"provider_type,omitempty"`
	ServiceType  string    `json:"service_type,omitempty"`
	Rating       float64   `json:"rating,omitempty"`
}
