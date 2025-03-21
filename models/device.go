// File: bloomify/models/device.go
package models

import "time"

type Device struct {
	DeviceID   string    `bson:"deviceId" json:"deviceId"`
	DeviceName string    `bson:"deviceName" json:"deviceName"`
	IP         string    `bson:"ip" json:"ip"`
	Location   string    `bson:"location" json:"location"`
	LastLogin  time.Time `bson:"lastLogin" json:"lastLogin"`
	Creator    bool      `bson:"creator" json:"creator"`
	TokenHash  string    `bson:"tokenHash" json:"-"`
}

// DeviceOTP holds temporary OTP data for device verification.
type DeviceOTP struct {
	UserID    string    `json:"userId"`
	DeviceID  string    `json:"deviceId"`
	OTP       string    `json:"otp"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type OTPPendingError struct {
	SessionID string
}
