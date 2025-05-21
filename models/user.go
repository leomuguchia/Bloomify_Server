// File: bloomify/models/user.go
package models

import "time"

type User struct {
	ID             string         `bson:"id" json:"id"`
	Username       string         `bson:"username" json:"username"`
	Email          string         `bson:"email" json:"email"`
	PhoneNumber    string         `bson:"phoneNumber" json:"phoneNumber"`
	Password       string         `bson:"-" json:"password,omitempty"`
	PasswordHash   string         `bson:"passwordHash" json:"-"`
	ProfileImage   string         `bson:"profileImage,omitempty" json:"profileImage,omitempty"`
	Preferences    []string       `bson:"preferences,omitempty" json:"preferences,omitempty"`
	Devices        []Device       `bson:"devices,omitempty" json:"devices,omitempty"`
	CreatedAt      time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time      `bson:"updatedAt" json:"updatedAt"`
	Rating         int            `bson:"rating" json:"rating,omitempty"`
	ActiveBookings []string       `bson:"activeBookings" json:"activeBookings,omitempty"`
	Notifications  []Notification `bson:"notifications" json:"notifications,omitempty"`
}
