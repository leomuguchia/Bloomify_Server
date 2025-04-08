package models

// BookingSession represents an active booking session.
type BookingSession struct {
	SessionID           string              `json:"sessionID"`
	ServicePlan         ServicePlan         `json:"servicePlan"`
	MatchedProviders    []ProviderDTO       `json:"matchedProviders"`
	SelectedProvider    string              `json:"selectedProvider,omitempty"`
	Availability        []AvailableSlot     `json:"availability,omitempty"`
	FullTimeSlotMapping map[string]TimeSlot `json:"fullTimeSlotMapping"`
	UserID              string              `json:"userID"`
	DeviceID            string              `json:"deviceID"`
	DeviceName          string              `json:"deviceName"`
	AvailabilityError   string              `json:"availabilityError,omitempty"`
	MaxAvailableDate    string              `json:"maxAvailableDate,omitempty"`
}

type BookingResponse struct {
	SessionID    string          `json:"sessionId,omitempty"`
	Providers    []ProviderDTO   `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}
