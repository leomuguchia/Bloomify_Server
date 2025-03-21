package models

type BookingSession struct {
	SessionID        string          `json:"sessionId"`
	ServicePlan      ServicePlan     `json:"servicePlan"`
	MatchedProviders []ProviderDTO   `json:"matchedProviders"`
	SelectedProvider string          `json:"selectedProviderId,omitempty"`
	Availability     []AvailableSlot `json:"availability,omitempty"`
	UserID           string          `json:"userId"`
	PaymentMethod    string          `json:"paymentMethod"`
	DeviceID         string          `json:"deviceId,omitempty"`
	UserAgent        string          `json:"userAgent,omitempty"`
}

type BookingResponse struct {
	SessionID    string          `json:"sessionId,omitempty"`
	Providers    []ProviderDTO   `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}

// InitiateSessionRequest defines the expected JSON payload for initiating a booking session.
type InitiateSessionRequest struct {
	ServicePlan
	DeviceID  string `json:"deviceId" binding:"required"`
	UserAgent string `json:"userAgent" binding:"required"`
}
