package models

// AIRequest is the payload coming from the frontend into /api/ai/chat.
type AIRequest struct {
	UserID      string   `json:"user_id"`               // unique user identifier
	Text        string   `json:"text"`                  // user’s message (voice→text or typed)
	LocationGeo GeoPoint `json:"locationGeo,omitempty"` // user’s current location
}

// AIAction is a single button/card action returned during booking steps.
type AIAction struct {
	Label       string `json:"label"`                 // text on the button
	Type        string `json:"type"`                  // e.g. "book", "select_provider", "select_slot", "chat"
	ServiceID   string `json:"service_id,omitempty"`  // when booking/recommend
	ProviderID  string `json:"provider_id,omitempty"` // when selecting a provider
	Description string `json:"description,omitempty"` // e.g. slot label or extra info
}

// AIResponse is what your handler returns to the frontend.
type AIResponse struct {
	Intent       string     `json:"intent"`      // "chat", "recommend", or "book"
	ServiceType  string     `json:"serviceType"` // the service being discussed/booked (ID)
	ResponseText string     `json:"response"`    // natural‐language reply
	Actions      []AIAction `json:"actions"`     // only non‐nil during booking steps
}

type AIContext struct {
	ServiceType   string `json:"serviceType"`
	BookingStep   int    `json:"bookingStep"`
	BookingSessID string `json:"bookingSessionId"`
}
