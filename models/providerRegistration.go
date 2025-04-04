package models

import "time"

type ProviderBasicRegistrationData struct {
	ProviderName string   `json:"providerName"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	PhoneNumber  string   `json:"phoneNumber"`
	Address      string   `json:"address,omitempty"`
	LocationGeo  GeoPoint `json:"locationGeo"`
}

type KYPVerificationData struct {
	LegalName   string `json:"legalName"`
	DocumentURL string `json:"documentUrl"`
	SelfieURL   string `json:"selfieUrl"`
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
