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
	user     user.UserService
	provider provider.ProviderService
}

func NewDefaultNotificationService(
	userSvc user.UserService,
	providerSvc provider.ProviderService,
) (*DefaultNotificationService, error) {
	if userSvc == nil || providerSvc == nil {
		return nil, fmt.Errorf("notification service initialization error: user or provider service is nil")
	}
	return &DefaultNotificationService{
		user:     userSvc,
		provider: providerSvc,
	}, nil
}

// SendUserPushNotification looks up a user’s FCM token and sends a push.
func (s *DefaultNotificationService) SendUserPushNotification(
	ctx context.Context,
	userID, title, body string,
	data map[string]string,
) error {
	u, err := s.user.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("SendUserPushNotification: could not find user %s: %w", userID, err)
	}
	token := u.FCMToken
	if token == "" {
		return fmt.Errorf("SendUserPushNotification: user %s has no FCM token", userID)
	}

	// 💥 Ensure role is set
	if _, ok := data["role"]; !ok {
		data["role"] = "user"
		fmt.Printf("⚠️ [SendUserPushNotification] 'role' not set, defaulting to 'user'\n")
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := utils.FCMClient.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("SendUserPushNotification: failed to send FCM message: %w", err)
	}

	fmt.Printf("SendUserPushNotification: successfully sent message: %s\n", response)
	return nil
}

func (s *DefaultNotificationService) SendProviderPushNotification(
	ctx context.Context,
	providerID, title, body string,
	data map[string]string,
) error {
	p, err := s.provider.GetProviderByID(ctx, providerID, true)
	if err != nil {
		return fmt.Errorf("SendProviderPushNotification: could not find provider %s: %w", providerID, err)
	}
	token := p.Security.FCMToken
	if token == "" {
		return fmt.Errorf("SendProviderPushNotification: provider %s has no FCM token", providerID)
	}

	// 💥 Ensure role is set
	if _, ok := data["role"]; !ok {
		data["role"] = "provider"
		fmt.Printf("⚠️ [SendProviderPushNotification] 'role' not set, defaulting to 'provider'\n")
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "high_priority",
				Sound:     "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority":  "10",
				"apns-push-type": "alert",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
				},
			},
		},
	}

	if _, err := utils.FCMClient.Send(ctx, msg); err != nil {
		return fmt.Errorf("SendProviderPushNotification: failed to send FCM message: %w", err)
	}

	return nil
}
