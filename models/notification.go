package models

import "time"

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data"`
	Sent      bool           `json:"sent"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Read      bool           `json:"read"`
	Message   string         `json:"message"`
}
