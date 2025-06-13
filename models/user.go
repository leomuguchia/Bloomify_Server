// File: bloomify/models/user.go
package models

import "time"

type User struct {
	ID               string            `bson:"id" json:"id"`
	Username         string            `bson:"username" json:"username"`
	Email            string            `bson:"email" json:"email"`
	PhoneNumber      string            `bson:"phoneNumber" json:"phoneNumber"`
	Password         string            `bson:"-" json:"password,omitempty"`
	FCMToken         string            `bson:"fcmToken" json:"fcmToken"`
	PasswordHash     string            `bson:"passwordHash" json:"-"`
	ProfileImage     string            `bson:"profileImage,omitempty" json:"profileImage,omitempty"`
	Preferences      []string          `bson:"preferences,omitempty" json:"preferences,omitempty"`
	Devices          []Device          `bson:"devices,omitempty" json:"devices,omitempty"`
	CreatedAt        time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time         `bson:"updatedAt" json:"updatedAt"`
	Rating           int               `bson:"rating" json:"rating,omitempty"`
	ActiveBookings   []string          `bson:"activeBookings" json:"activeBookings,omitempty"`
	Notifications    []Notification    `bson:"notifications" json:"notifications,omitempty"`
	Location         GeoPoint          `bson:"location" json:"location,omitempty"`
	BookingHistory   []string          `bson:"bookingHistory" json:"bookingHistory,omitempty"`
	LastBookingTime  time.Time         `bson:"lastBookingTime" json:"lastBookingTime,omitempty"`
	SafetySettings   SafetySettings    `bson:"safetySettings,omitempty" json:"safetySettings,omitempty"`
	TrustedProviders []TrustedProvider `bson:"trustedProviders,omitempty" json:"trustedProviders,omitempty"`
}

type UserMinimal struct {
	ID           string   `bson:"id" json:"id"`
	Username     string   `bson:"username" json:"username"`
	ProfileImage string   `bson:"profileImage,omitempty" json:"profileImage,omitempty"`
	Rating       int      `bson:"rating" json:"rating,omitempty"`
	Location     GeoPoint `bson:"location" json:"location,omitzero"` // only include location if mode is provider-to-user
	PhoneNumber  string   `bson:"phoneNumber" json:"phoneNumber"`
}

type SafetySettings struct {
	NoShowThresholdMinutes int    `bson:"noShowThresholdMinutes" json:"noShowThresholdMinutes"`
	SafetyReminderMinutes  int    `bson:"safetyReminderMinutes" json:"safetyReminderMinutes"`
	RequireInsured         bool   `bson:"requireInsured" json:"requireInsured"`
	AlertChannel           string `bson:"alertChannel" json:"alertChannel"` // "sms", "push" or "both"
	EmailUpdates           bool   `bson:"emailUpdates" json:"emailUpdates"` // Whether to send email updates
}

type TrustedProvider struct {
	ProviderID   string    `bson:"providerId" json:"providerId"`
	ProviderName string    `bson:"providerName" json:"providerName"`
	ServiceType  string    `bson:"serviceType" json:"serviceType"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
}

type UserUpdateRequest struct {
	ID                    *string            `bson:"id" json:"id"`
	Username              *string            `json:"username,omitempty"`
	Email                 *string            `json:"email,omitempty"`
	PhoneNumber           *string            `json:"phoneNumber,omitempty"`
	FCMToken              *string            `json:"fcmToken,omitempty"`
	ProfileImage          *string            `json:"profileImage,omitempty"`
	Preferences           *[]string          `json:"preferences,omitempty"`
	Devices               *[]Device          `json:"devices,omitempty"`
	Rating                *int               `json:"rating,omitempty"`
	ActiveBookings        *[]string          `json:"activeBookings,omitempty"`
	Notifications         *[]Notification    `json:"notifications,omitempty"`
	Location              *GeoPoint          `json:"location,omitempty"`
	BookingHistory        *[]string          `json:"bookingHistory,omitempty"`
	LastBookingTime       *time.Time         `json:"lastBookingTime,omitempty"`
	SafetySettings        *SafetySettings    `json:"safetySettings,omitempty"`
	TrustedProviders      *[]TrustedProvider `json:"trustedProviders,omitempty"`
	UpdatedAt             *time.Time         `json:"updatedAt,omitempty"`
	MarkNotificationsRead *[]string          `json:"markNotificationsRead,omitempty"`
	RemoveNotifications   *[]string          `json:"removeNotifications,omitempty"`
}
