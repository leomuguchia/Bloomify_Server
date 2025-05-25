package models

import "time"

// --- PaymentRequest & Invoice ---
type PaymentRequest struct {
	UserID          string
	Amount          float64
	Method          string // "cash" or "card"
	Currency        string
	Metadata        map[string]string
	PaymentIntentID string
	Action          string
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

type PublicInvoice struct {
	InvoiceID string    `json:"invoiceId"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Method    string    `json:"method"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ToPublicInvoice(inv Invoice) PublicInvoice {
	return PublicInvoice{
		InvoiceID: inv.InvoiceID,
		Amount:    inv.Amount,
		Currency:  inv.Currency,
		Status:    inv.Status,
		Method:    inv.Method,
		CreatedAt: inv.CreatedAt,
		UpdatedAt: inv.UpdatedAt,
	}
}

type PaymentIntentRequest struct {
	Amount   float64 `json:"amount" binding:"required"`   // e.g., 10.00
	Currency string  `json:"currency" binding:"required"` // e.g., "usd"
}

type UserPayment struct {
	PaymentMethod string `json:"paymentMethod" binding:"required"` //cash or card via stripe
	Currency      string `json:"currency" binding:"required"`

	// stripe required details, omit if using cash
	PaymentIntentId string `json:"paymentIntentId,omitempty"`
}
