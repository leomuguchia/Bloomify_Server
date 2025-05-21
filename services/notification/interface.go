package notification

import (
	"bloomify/models"
	"context"
)

type NotificationService interface {
	// PublishEvent just records an event (e.g. "booking.finalized", payload)
	PublishEvent(ctx context.Context, eventType string, payload map[string]interface{}) error

	// Optionally, send a one‑off notification immediately
	SendNotification(ctx context.Context, notif *models.Notification) error
}
