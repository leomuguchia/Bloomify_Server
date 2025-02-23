// models/user.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a platform user.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `json:"name"`
	Email        string         `gorm:"uniqueIndex" json:"email"`
	PasswordHash string         `json:"password_hash"` // Store hashed passwords
	PhoneNumber  string         `json:"phone_number"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
