package models

type BookingSession struct {
	SessionID        string          `json:"sessionId"`
	ServicePlan      ServicePlan     `json:"servicePlan"`
	MatchedProviders []ProviderDTO   `json:"matchedProviders"`
	SelectedProvider string          `json:"selectedProviderId,omitempty"`
	Availability     []AvailableSlot `json:"availability,omitempty"`
	UserID           string          `json:"user_id"`
	PaymentMethod    string          `json:"payment_method"`
	DeviceID         string          `json:"device_id,omitempty"`
	UserAgent        string          `json:"user_agent,omitempty"`
}

type BookingResponse struct {
	SessionID    string          `json:"sessionID,omitempty"`
	Providers    []ProviderDTO   `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}

// InitiateSessionRequest defines the expected JSON payload for initiating a booking session.
type InitiateSessionRequest struct {
	ServicePlan
	DeviceID  string `json:"device_id" binding:"required"`
	UserAgent string `json:"user_agent" binding:"required"`
}
