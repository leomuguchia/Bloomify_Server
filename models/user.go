package models

import "time"

// User represents a customer who uses the platform to connect with service providers.
type User struct {
	ID           string    `bson:"id" json:"id"`
	Username     string    `bson:"username" json:"username"`
	Email        string    `bson:"email" json:"email"`
	PhoneNumber  string    `bson:"phone_number" json:"phoneNumber"`
	Password     string    `bson:"-" json:"password,omitempty"`                           // Transient field for registration.
	PasswordHash string    `bson:"password_hash" json:"-"`                                // Stored hashed password.
	TokenHash    string    `bson:"token_hash" json:"-"`                                   // Stored token hash.
	ProfileImage string    `bson:"profile_image,omitempty" json:"profileImage,omitempty"` // Optional profile image URL.
	Preferences  []string  `bson:"preferences,omitempty" json:"preferences,omitempty"`    // Preferred services.
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
	Rating       int       `bson:"rating" json:"rating,omitempty"`
}
