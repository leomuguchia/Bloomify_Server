package models

import "time"

type UserBasicRegistrationData struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	PhoneNumber  string `json:"phoneNumber" binding:"required"`
	ProfileImage string `json:"profileImage"`
}

type UserRegistrationRequest struct {
	Step         string                     `json:"step"`
	SessionID    string                     `json:"sessionID,omitempty"`
	OTP          string                     `json:"otp,omitempty"`
	BasicData    *UserBasicRegistrationData `json:"basicData,omitempty"`
	Preferences  []string                   `json:"preferences,omitempty"`
	EmailUpdates bool                       `json:"emailUpdates,omitempty"`
}

type UserRegistrationSession struct {
	TempID        string                     `json:"tempId" bson:"tempId"`
	BasicData     *UserBasicRegistrationData `json:"basicData,omitempty" bson:"basicData,omitempty"`
	OTPStatus     string                     `json:"otpStatus" bson:"otpStatus"`
	CreatedAt     time.Time                  `json:"createdAt" bson:"createdAt"`
	LastUpdatedAt time.Time                  `json:"lastUpdatedAt" bson:"lastUpdatedAt"`
	Devices       []Device                   `json:"devices,omitempty" bson:"devices,omitempty"`
}
