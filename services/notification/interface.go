package notification

import (
	"context"
	"fmt"

	"bloomify/services/provider"
	"bloomify/services/user"
	"bloomify/utils"

	"firebase.google.com/go/v4/messaging"
)

// NotificationService defines methods for sending FCM pushes.
type NotificationService interface {
	SendUserPushNotification(ctx context.Context, userID, title, body string, data map[string]string) error
	SendProviderPushNotification(ctx context.Context, providerID, title, body string, data map[string]string) error
}

// DefaultNotificationService is the production implementation.
type DefaultNotificationService struct {
	User     user.UserService
	Provider provider.ProviderService
}

// SendUserPushNotification looks up a user’s FCM token and sends a push.
func (s *DefaultNotificationService) SendUserPushNotification(
	ctx context.Context,
	userID, title, body string,
	data map[string]string,
) error {
	// 1. Fetch the user (including their FCMToken)
	u, err := s.User.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("SendUserPushNotification: could not find user %s: %w", userID, err)
	}
	token := u.FCMToken
	if token == "" {
		return fmt.Errorf("SendUserPushNotification: user %s has no FCM token", userID)
	}

	// 2. Build the FCM message
	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	// 3. Send via utils.FCMClient (initialized in utils.Init())
	response, err := utils.FCMClient.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("SendUserPushNotification: failed to send FCM message: %w", err)
	}

	// 4. Log the response ID (optional)
	fmt.Printf("SendUserPushNotification: successfully sent message: %s\n", response)
	return nil
}

// SendProviderPushNotification looks up a provider’s FCM token and sends a push.
func (s *DefaultNotificationService) SendProviderPushNotification(
	ctx context.Context,
	providerID, title, body string,
	data map[string]string,
) error {
	// 1. Fetch the provider (including their FCMToken)
	p, err := s.Provider.GetProviderByID(ctx, providerID, true)
	if err != nil {
		return fmt.Errorf("SendProviderPushNotification: could not find provider %s: %w", providerID, err)
	}
	token := p.Security.FCMToken
	if token == "" {
		return fmt.Errorf("SendProviderPushNotification: provider %s has no FCM token", providerID)
	}

	// 2. Build the FCM message
	msg := &messaging.Message{
		Token: token,

		// This is what lets FCM/APNS/Android auto‐post a banner
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},

		// Optional data payload you can still read in onMessage
		Data: data,

		// AND on Android, bump it to HIGH priority and point at a high‐importance channel:
		Android: &messaging.AndroidConfig{
			Priority: "high", // heads-up
			Notification: &messaging.AndroidNotification{
				ChannelID: "high_priority", // make sure this channel exists on the device
				Sound:     "default",
			},
		},

		// AND on iOS, request an immediate “alert” via APNS headers:
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority":  "10",    // immediate delivery
				"apns-push-type": "alert", // required on iOS13+
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
					// Badge: &badgeCount,    // optional badge number
				},
			},
		},
	}

	// 3. Send via utils.FCMClient
	if _, err := utils.FCMClient.Send(ctx, msg); err != nil {
		return fmt.Errorf("SendProviderPushNotification: failed to send FCM message: %w", err)
	}

	return nil
}
