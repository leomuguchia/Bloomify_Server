package models

import "time"

type ProviderBasicRegistrationData struct {
	ProviderName    string   `json:"providerName" binding:"required"`
	ProviderType    string   `json:"providerType" binding:"required"`
	ProfileImageUrl string   `json:"profileImageUrl,omitempty"`
	Email           string   `json:"email" binding:"required"`
	Password        string   `json:"password" binding:"required"`
	PhoneNumber     string   `json:"phoneNumber" binding:"required"`
	Address         string   `json:"address,omitempty"`
	LocationGeo     GeoPoint `json:"locationGeo" binding:"required"`
	Description     string   `json:"description" binding:"required"`
}

type KYPVerificationData struct {
	Type         string `json:"type"` // "freelancer" or "business"
	LegalName    string `json:"legalName"`
	DocumentURL  string `json:"documentUrl"`
	DocumentType string `json:"documentType"` // e.g., "passport", "license", "taxID"
	SelfieURL    string `json:"selfieUrl"`    // required for freelancer, optional for business contact person
	ContactName  string `json:"contactName"`  // optional for businesses
	ContactEmail string `json:"contactEmail"` // optional for businesses
}

// RegistrationSession holds all transient data during multi‑step registration.
type ProviderRegistrationSession struct {
	TempID             string                        `json:"tempId"`                     // Unique session ID.
	BasicData          ProviderBasicRegistrationData `json:"basicData,omitempty"`        // Data from Step 1.
	KYPData            KYPVerificationData           `json:"kypData,omitempty"`          // Data from Step 2.
	ServiceCatalogue   ServiceCatalogue              `json:"serviceCatalogue,omitempty"` // Data from Step 3.
	OTPStatus          string                        `json:"otpStatus"`                  // e.g., "pending", "verified"
	VerificationStatus string                        `json:"verificationStatus"`         // e.g., "pending", "verified"
	VerificationLevel  string                        ` json:"verificationLevel,omitempty"`
	CreatedAt          time.Time                     `json:"createdAt"`
	LastUpdatedAt      time.Time                     `json:"lastUpdatedAt"`
	Devices            []Device                      `json:"devices,omitempty"` // Captured device(s) during registration.
}

// RegistrationRequest is the composite request payload for multi‑step registration.
// The client includes the "step" field to indicate which part of the flow is being executed.
type ProviderRegistrationRequest struct {
	Step             string                         `json:"step"`                       // "basic", "otp", "kyp", or "catalogue"
	SessionID        string                         `json:"sessionID,omitempty"`        // Required for steps "otp", "kyp", and "catalogue"
	OTP              string                         `json:"otp,omitempty"`              // Used only in the OTP verification step.
	BasicData        *ProviderBasicRegistrationData `json:"basicData,omitempty"`        // For step "basic"
	KYPData          *KYPVerificationData           `json:"kypData,omitempty"`          // For step "kyp"
	ServiceCatalogue *ServiceCatalogue              `json:"serviceCatalogue,omitempty"` // For step "catalogue"
}

type ProviderDTO struct {
	ID               string           `json:"id"`
	Profile          Profile          `json:"profile"`
	ServiceCatalogue ServiceCatalogue `json:"serviceCatalogue"`
	LocationGeo      GeoPoint         `json:"locationGeo"`
	Preferred        bool             `json:"preferred"`
	Proximity        float64          `json:"proximity"`
	Icon             string           `json:"icon,omitempty"`
}

type ProviderAuthResponse struct {
	ID               string           `json:"id"`
	Token            string           `json:"token"`
	Profile          Profile          `json:"profile"`
	CreatedAt        time.Time        `json:"createdAt"`
	ServiceCatalogue ServiceCatalogue `json:"serviceCatalogue"`
}
