package utils

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// generateSecureOTP generates a secure random OTP of the specified length.
// It returns a base32 encoded string (without padding) truncated to the desired length.
func generateSecureOTP(length int) (string, error) {
	numBytes := (length*5 + 7) / 8 // Calculate the required number of bytes.
	randomBytes := make([]byte, numBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	otp := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	if len(otp) > length {
		otp = otp[:length]
	}
	return otp, nil
}

// SendWhatsAppMessage sends a WhatsApp message to the given phone number.
// Replace the body of this function with your actual integration with WhatsApp's API.
func SendWhatsAppMessage(phoneNumber, message string) error {
	// For example, you could use an HTTP client to call your WhatsApp API endpoint:
	// resp, err := http.Post("https://api.yourwhatsappprovider.com/send", "application/json", payloadReader)
	// Handle response and errors accordingly.
	// For now, we log the outgoing message.
	GetLogger().Sugar().Infof("Sending WhatsApp message to %s: %s", phoneNumber, message)
	return nil
}

// InitiateDeviceOTP generates an OTP, stores it in Redis with a 5-minute TTL,
// sends it via WhatsApp, and also stores the OTP in the Test Redis with a key based on sessionID.
func InitiateDeviceOTP(userID, deviceID, phoneNumber string) error {
	// Generate a secure 6-character OTP.
	otp, err := generateSecureOTP(6)
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}
	ttl := 5 * time.Minute
	otpKey := fmt.Sprintf("otp:%s:%s", userID, deviceID)

	ctx := context.Background()
	client := GetOTPCacheClient()
	if client == nil {
		return fmt.Errorf("OTP cache client not initialized")
	}

	// Store the OTP in the normal OTP cache with a TTL of 5 minutes.
	if err := client.Set(ctx, otpKey, otp, ttl).Err(); err != nil {
		GetLogger().Error("Failed to cache OTP", zap.Error(err))
		return fmt.Errorf("failed to initiate device OTP")
	}

	// Compose the message to send.
	message := fmt.Sprintf("Your Bloomify OTP is: %s. It expires in 5 minutes.", otp)
	// Send the OTP via WhatsApp.
	if err := SendWhatsAppMessage(phoneNumber, message); err != nil {
		GetLogger().Error("Failed to send OTP via WhatsApp", zap.Error(err))
		return fmt.Errorf("failed to send OTP")
	}

	// Additionally, store the OTP in the Test Redis using sessionID as key.
	testKey := fmt.Sprintf("session:%s", userID)
	testClient := GetTestCacheClient()
	if testClient == nil {
		GetLogger().Error("Test cache client not initialized")
	} else {
		if err := testClient.Set(ctx, testKey, otp, ttl).Err(); err != nil {
			GetLogger().Error("Failed to cache OTP in Test Redis", zap.Error(err))
		} else {
			GetLogger().Sugar().Infof("Stored OTP %s in Test Redis under key %s", otp, testKey)
		}
	}

	GetLogger().Sugar().Infof("Sent OTP %s to phone %s for user %s, device %s (expires in %v)", otp, phoneNumber, userID, deviceID, ttl)
	return nil
}

// VerifyDeviceOTPRecord retrieves the stored OTP from Redis and compares it to the provided OTP.
// If they match, it deletes the OTP from the cache.
func VerifyDeviceOTPRecord(userID, deviceID, providedOTP string) error {
	otpKey := fmt.Sprintf("otp:%s:%s", userID, deviceID)
	ctx := context.Background()
	client := GetOTPCacheClient()
	if client == nil {
		return fmt.Errorf("OTP cache client not initialized")
	}

	storedOTP, err := client.Get(ctx, otpKey).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("OTP not found or expired")
		}
		return fmt.Errorf("failed to retrieve OTP: %w", err)
	}

	if storedOTP != providedOTP {
		return fmt.Errorf("OTP does not match")
	}

	// Delete the OTP after successful verification.
	if err := client.Del(ctx, otpKey).Err(); err != nil {
		GetLogger().Error("Failed to delete OTP after verification", zap.Error(err))
	}
	return nil
}
