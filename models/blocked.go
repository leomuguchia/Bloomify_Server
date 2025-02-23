// models/blocked.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// Blocked represents a blocked time interval for a provider.
type Blocked struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ProviderID  string         `json:"provider_id"`
	Date        string         `json:"date"` // YYYY-MM-DD
	StartMinute int            `json:"start_minute"`
	EndMinute   int            `json:"end_minute"`
	Reason      string         `json:"reason"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
