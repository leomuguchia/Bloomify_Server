// File: services/booking/bookingUpdates.go
package booking

import (
	"bloomify/models"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func (se *DefaultSchedulingEngine) NotifyUserWithBookingStatus(
	provider models.Provider,
	booking *models.Booking,
	paymentCaptureFailed bool,
) bool {
	user, err := se.UserService.GetUserByID(booking.UserID)
	if err != nil {
		log.Printf("[NotifyUserWithBookingStatus] Failed to fetch user: %v", err)
		return false
	}

	user.ActiveBookings = append(user.ActiveBookings, booking.ID)

	var title, message string
	if paymentCaptureFailed {
		title = "Payment Needed ❗"
		message = fmt.Sprintf("Your booking with %s is confirmed but payment failed. "+
			"Please update your payment method for %s %.2f to secure your appointment.",
			provider.Profile.ProviderName, booking.UserPayment.Currency, booking.TotalPrice)
	} else {
		title = "Booking Confirmed ✅"
		message = fmt.Sprintf("Your appointment with %s on %s is confirmed!",
			provider.Profile.ProviderName, booking.Date)

		if booking.UserPayment.PaymentMethod == "cash" {
			message += fmt.Sprintf(" Please bring %s %.2f when you arrive.",
				booking.UserPayment.Currency, booking.TotalPrice)
		} else {
			message += fmt.Sprintf(" %s %.2f has been processed.",
				booking.UserPayment.Currency, booking.TotalPrice)
		}
	}

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    "booking_confirmation",
		Message: message,
		Data: map[string]any{
			"bookingDate":  booking.Date,
			"providerName": provider.Profile.ProviderName,
			"status":       booking.Status,
			"amount":       booking.TotalPrice,
			"currency":     booking.UserPayment.Currency,
		},
		CreatedAt: time.Now(),
		Read:      false,
	}

	user.Notifications = append(user.Notifications, notification)
	user.UpdatedAt = time.Now()

	if _, err := se.UserService.UpdateUser(*user); err != nil {
		log.Printf("[NotifyUserWithBookingStatus] Failed to update user: %v", err)
		return false
	}

	// Determine geo for push notification
	var locationGeo *models.GeoPoint
	if booking.Mode == "in_store" && len(provider.Profile.LocationGeo.Coordinates) == 2 {
		locationGeo = &provider.Profile.LocationGeo
	}

	go func() {
		data := map[string]string{
			"notify":     "Booking Confirmation",
			"date":       booking.Date,
			"service by": provider.Profile.ProviderName,
			"status":     booking.Status,
		}
		if locationGeo != nil {
			data["longitude"] = fmt.Sprintf("%f", locationGeo.Coordinates[0])
			data["latitude"] = fmt.Sprintf("%f", locationGeo.Coordinates[1])
		}

		err := se.Notification.SendUserPushNotification(
			context.Background(),
			user.ID,
			title,
			message,
			data,
		)
		if err != nil {
			log.Printf("[PushNotification] Failed to send user push notification: %v", err)
		}
	}()

	return true
}

func (se *DefaultSchedulingEngine) UpdateProviderWithBookingNotification(
	provider *models.Provider,
	booking *models.Booking,
	slot models.TimeSlot,
	used int,
) bool {
	remaining := slot.Capacity - used

	message := fmt.Sprintf(
		"New booking made for %s. Remaining capacity in slot [%d–%d]: %d",
		booking.Date, slot.Start, slot.End, remaining,
	)

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    "booking_notice",
		Message: message,
		Data: map[string]any{
			"bookingId":         booking.ID,
			"remainingCapacity": remaining,
			"slotId":            slot.ID,
			"date":              booking.Date,
		},
		CreatedAt: time.Now(),
		Read:      false,
	}

	user, err := se.UserService.GetUserByID(booking.UserID)
	if err != nil {
		log.Printf("[UpdateProviderWithBookingNotification] Failed to fetch user: %v", err)
		return false
	}

	activeBooking := models.ActiveBookingDTO{
		BookingID: booking.ID,
		CreatedAt: booking.CreatedAt,
		End:       booking.End,
		User: models.UserMinimal{
			ID:           booking.UserID,
			Username:     user.Username,
			ProfileImage: user.ProfileImage,
			Rating:       user.Rating,
			PhoneNumber:  user.PhoneNumber,
			Location:     user.Location,
		},
	}

	now := time.Now()

	updateDoc := bson.M{
		"$push": bson.M{
			"notifications":  notification,
			"activeBookings": activeBooking,
		},
		"$set": bson.M{
			"updatedAt": now,
		},
	}

	err = se.ProviderRepo.UpdateWithDocument(provider.ID, updateDoc)
	if err != nil {
		log.Printf("[UpdateProviderWithBookingNotification] Failed to update provider: %v", err)
		return false
	}

	// Send push notification (non-blocking)
	go func() {
		err := se.Notification.SendProviderPushNotification(
			context.Background(),
			provider.ID,
			"New Booking 🧑‍💼",
			message,
			map[string]string{
				"type":              "booking_notice",
				"bookingId":         booking.ID,
				"remainingCapacity": fmt.Sprintf("%d", remaining),
				"slotId":            slot.ID,
				"date":              booking.Date,
			},
		)
		if err != nil {
			log.Printf("[PushNotification] Failed to send provider notification: %v", err)
		}
	}()

	return true
}
