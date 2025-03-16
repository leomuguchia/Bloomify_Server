// File: bloomify/models/device.go
package models

import "time"

// Device represents a device associated with a user.
type Device struct {
	DeviceID   string    `bson:"device_id" json:"device_id"`
	DeviceName string    `bson:"device_name" json:"device_name"`
	IP         string    `bson:"ip" json:"ip"`
	Location   string    `bson:"location" json:"location"`
	LastLogin  time.Time `bson:"last_login" json:"last_login"`
	Creator    bool      `bson:"creator" json:"creator"`
}

// DeviceOTP holds temporary OTP data for device verification.
type DeviceOTP struct {
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	OTP       string    `json:"otp"`
	ExpiresAt time.Time `json:"expires_at"`
}

type OTPPendingError struct {
	SessionID string
}
