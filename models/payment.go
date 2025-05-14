package models

import "time"

// --- PaymentRequest & Invoice ---
type PaymentRequest struct {
	UserID      string
	Amount      float64
	Method      string // "cash" or "card"
	Currency    string
	Idempotency string
	Metadata    map[string]string
	Description string
}

type Invoice struct {
	InvoiceID string
	UserID    string
	Amount    float64
	Currency  string
	Status    string
	Method    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Retries   int
	PaymentID string
	Error     string
}
