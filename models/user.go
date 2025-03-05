package models

import "time"

// User represents a customer who uses the platform to connect with service providers.
type User struct {
	ID           string `bson:"id" json:"id"`
	Username     string `bson:"username" json:"username"` // Public username or display name.
	Email        string `bson:"email" json:"email"`
	PhoneNumber  string `bson:"phone_number" json:"phone_number"`
	Password     string `bson:"-" json:"password,omitempty"` // Transient field for registration.
	PasswordHash string `bson:"password_hash" json:"-"`      // Stored hashed password.
	TokenHash    string `bson:"token_hash" json:"-"`         // Stored token hash.
	// Additional fields as needed (e.g., Address, ProfilePicture, etc.)
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
