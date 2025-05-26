// File: bloomify/models/user.go
package models

import "time"

type User struct {
	ID              string         `bson:"id" json:"id"`
	Username        string         `bson:"username" json:"username"`
	Email           string         `bson:"email" json:"email"`
	PhoneNumber     string         `bson:"phoneNumber" json:"phoneNumber"`
	Password        string         `bson:"-" json:"password,omitempty"`
	FCMToken        string         `bson:"fcmToken" json:"fcmToken"`
	PasswordHash    string         `bson:"passwordHash" json:"-"`
	ProfileImage    string         `bson:"profileImage,omitempty" json:"profileImage,omitempty"`
	Preferences     []string       `bson:"preferences,omitempty" json:"preferences,omitempty"`
	Devices         []Device       `bson:"devices,omitempty" json:"devices,omitempty"`
	CreatedAt       time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time      `bson:"updatedAt" json:"updatedAt"`
	Rating          int            `bson:"rating" json:"rating,omitempty"`
	ActiveBookings  []string       `bson:"activeBookings" json:"activeBookings,omitempty"`
	Notifications   []Notification `bson:"notifications" json:"notifications,omitempty"`
	Location        GeoPoint       `bson:"location" json:"location,omitempty"`
	BookingHistory  []string       `bson:"bookingHistory" json:"bookingHistory,omitempty"`
	LastBookingTime time.Time      `bson:"lastBookingTime" json:"lastBookingTime,omitempty"`
	SafetySettings  SafetySettings `bson:"safetySettings,omitempty" json:"safetySettings,omitempty"`
}

type UserMinimal struct {
	ID           string   `bson:"id" json:"id"`
	Username     string   `bson:"username" json:"username"`
	ProfileImage string   `bson:"profileImage,omitempty" json:"profileImage,omitempty"`
	Rating       int      `bson:"rating" json:"rating,omitempty"`
	Location     GeoPoint `bson:"location" json:"location,omitempty"` // only include location if mode is provider-to-user
	PhoneNumber  string   `bson:"phoneNumber" json:"phoneNumber"`
}

type SafetySettings struct {
	NoShowThresholdMinutes int    `bson:"noShowThresholdMinutes" json:"noShowThresholdMinutes"`
	SafetyReminderMinutes  int    `bson:"safetyReminderMinutes" json:"safetyReminderMinutes"`
	RequireInsured         bool   `bson:"requireInsured" json:"requireInsured"`
	AlertChannel           string `bson:"alertChannel" json:"alertChannel"` // "sms", "push" or "both"
	EmailUpdates           bool   `bson:"emailUpdates" json:"emailUpdates"` // Whether to send email updates
}
