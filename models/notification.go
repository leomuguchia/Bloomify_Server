package models

import "time"

type Notification struct {
	ID        string         `json:"id"`     // UUID
	UserID    string         `json:"userId"` // whom to notify
	Type      string         `json:"type"`   // e.g., "booking.finalized"
	Title     string         `json:"title"`  // e.g., "Your booking is confirmed!"
	Body      string         `json:"body"`   // human‐readable message
	Data      map[string]any `json:"data"`   // optional extra payload (e.g. bookingID)
	Sent      bool           `json:"sent"`   // whether it was delivered successfully
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Read      bool           `json:"read"` // only relevant for in_app
	Message   string         `json:"message"`
}
