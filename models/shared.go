package models

import "time"

type Reminder struct {
	ID       string    `bson:"id" json:"id"`
	Title    string    `bson:"title" json:"title"`
	Body     string    `bson:"body" json:"body"`
	FireDate time.Time `bson:"fireDate" json:"fireDate"`
	Sent     bool      `bson:"sent" json:"sent"`
}

type ReminderPayload struct {
	ID         string `json:"id"`         // userId or providerId
	ReminderID string `json:"reminderId"` // optional
	Title      string `json:"title"`
	Body       string `json:"body"`
	FireDate   string `json:"fireDate"` // optional
	Target     string `json:"target"`   // "user" or "provider"
}
