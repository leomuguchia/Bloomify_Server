package models

// BookingSession represents an active booking session.
type BookingSession struct {
	SessionID           string              `json:"sessionID"`
	ServicePlan         ServicePlan         `json:"servicePlan"`
	MatchedProviders    []ProviderDTO       `json:"matchedProviders"`
	SelectedProvider    string              `json:"selectedProvider,omitempty"`
	Availability        []AvailableSlot     `json:"availability,omitempty"`
	FullTimeSlotMapping map[string]TimeSlot `json:"-"` // Map availableSlot.ID -> full TimeSlot (do not expose externally)
	UserID              string              `json:"userID"`
	DeviceID            string              `json:"deviceID"`
	DeviceName          string              `json:"deviceName"`
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
