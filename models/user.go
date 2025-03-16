// File: bloomify/models/user.go
package models

import "time"

type User struct {
	ID           string    `bson:"id" json:"id"`
	Username     string    `bson:"username" json:"username"`
	Email        string    `bson:"email" json:"email"`
	PhoneNumber  string    `bson:"phone_number" json:"phoneNumber"`
	Password     string    `bson:"-" json:"password,omitempty"`
	PasswordHash string    `bson:"password_hash" json:"-"`
	TokenHash    string    `bson:"token_hash" json:"-"`
	ProfileImage string    `bson:"profile_image,omitempty" json:"profileImage,omitempty"`
	Preferences  []string  `bson:"preferences,omitempty" json:"preferences,omitempty"`
	Devices      []Device  `bson:"devices,omitempty" json:"devices,omitempty"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
	Rating       int       `bson:"rating" json:"rating,omitempty"`
}
