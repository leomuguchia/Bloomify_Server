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

func formatBookingDateTime(dateStr string, minutesFromMidnight int) (string, error) {
	bookingDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}
	bookingTime := bookingDate.Add(time.Duration(minutesFromMidnight) * time.Minute)
	return bookingTime.Format("2 January, 3:04 PM"), nil
}

func (se *DefaultSchedulingEngine) NotifyUserWithBookingStatus(
	provider models.Provider,
	booking *models.Booking,
	paymentCaptureFailed bool,
) bool {
	user, err := se.UserService.GetUserByID(booking.UserID)
	if err != nil {
		log.Printf("[NotifyUserWithBookingStatus] Failed to fetch user %s: %v", booking.UserID, err)
		return false
	}

	user.ActiveBookings = append(user.ActiveBookings, booking.ID)

	formattedDateTime, err := formatBookingDateTime(booking.Date, booking.Start)
	if err != nil {
		log.Printf("[NotifyUserWithBookingStatus] Failed to format booking time: %v", err)
		return false
	}

	var title, message, notificationType string
	var actionRequired bool

	if paymentCaptureFailed {
		title = "Payment Needed for Your Booking"
		message = fmt.Sprintf("Your booking with %s is confirmed but we couldn't process your payment. Please update your payment method to secure your appointment on %s.",
			provider.Profile.ProviderName, formattedDateTime)
		notificationType = "payment_required"
		actionRequired = true
	} else {
		title = "Booking Confirmed!"
		notificationType = "booking_confirmed"
		actionRequired = false

		message = fmt.Sprintf("Your appointment with %s on %s has been confirmed.",
			provider.Profile.ProviderName, formattedDateTime)

		if booking.UserPayment.PaymentMethod == "cash" {
			message += fmt.Sprintf(" Please have %s %.2f in cash on arrival.",
				booking.UserPayment.Currency, booking.TotalPrice)
		} else {
			message += fmt.Sprintf(" We've successfully processed your payment of %s %.2f.",
				booking.UserPayment.Currency, booking.TotalPrice)
		}
	}

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    notificationType,
		Title:   title,
		Message: message,
		Data: map[string]any{
			"bookingId": booking.ID,
			"date":      booking.Date,
			"time":      booking.Start,
			"dateTime":  formattedDateTime,
			"provider": map[string]any{
				"id":    provider.ID,
				"name":  provider.Profile.ProviderName,
				"image": provider.Profile.ProfileImage,
			},
			"amount":         booking.TotalPrice,
			"currency":       booking.UserPayment.Currency,
			"status":         booking.Status,
			"actionRequired": actionRequired,
		},
		CreatedAt: time.Now(),
		Read:      false,
	}

	user.Notifications = append(user.Notifications, notification)
	user.UpdatedAt = time.Now()

	updateReq := models.UserUpdateRequest{
		ID:             &user.ID,
		ActiveBookings: &user.ActiveBookings,
		Notifications:  &user.Notifications,
		UpdatedAt:      &user.UpdatedAt,
	}
	if _, err := se.UserService.UpdateUser(updateReq); err != nil {
		log.Printf("[NotifyUserWithBookingStatus] Failed to update user %s: %v", user.ID, err)
		return false
	}

	// Determine geo for push notification
	var locationGeo *models.GeoPoint
	if booking.Mode == "in_store" && len(provider.Profile.LocationGeo.Coordinates) == 2 {
		locationGeo = &provider.Profile.LocationGeo
	}

	go func() {
		data := map[string]string{
			"type":           notificationType,
			"bookingId":      booking.ID,
			"providerId":     provider.ID,
			"providerName":   provider.Profile.ProviderName,
			"date":           booking.Date,
			"time":           fmt.Sprintf("%d", booking.Start),
			"dateTime":       formattedDateTime,
			"status":         booking.Status,
			"actionRequired": fmt.Sprintf("%t", actionRequired),
		}
		if locationGeo != nil {
			data["locationLongitude"] = fmt.Sprintf("%f", locationGeo.Coordinates[0])
			data["locationLatitude"] = fmt.Sprintf("%f", locationGeo.Coordinates[1])
		}

		err := se.Notification.SendUserPushNotification(
			context.Background(),
			user.ID,
			title,
			message,
			data,
		)
		if err != nil {
			log.Printf("[PushNotification] Failed to send user %s push notification: %v", user.ID, err)
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
	startHour := slot.Start / 60
	startMin := slot.Start % 60
	endHour := slot.End / 60
	endMin := slot.End % 60

	startTime := time.Date(0, 1, 1, startHour, startMin, 0, 0, time.UTC)
	endTime := time.Date(0, 1, 1, endHour, endMin, 0, 0, time.UTC)

	timeRange := fmt.Sprintf("%s - %s", startTime.Format("3:04 PM"), endTime.Format("3:04 PM"))

	user, err := se.UserService.GetUserByID(booking.UserID)
	if err != nil {
		log.Printf("[UpdateProviderWithBookingNotification] Failed to fetch user %s: %v", booking.UserID, err)
		return false
	}

	formattedDateTime, err := formatBookingDateTime(booking.Date, booking.Start)
	if err != nil {
		log.Printf("[UpdateProviderWithBookingNotification] Failed to format booking time: %v", err)
		return false
	}

	title := "New Booking Received"
	message := fmt.Sprintf(
		"%s booked your service for %s at %s. %d %s remaining in this schedule slot.",
		user.Username, formattedDateTime, timeRange, remaining, booking.UnitType,
	)

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    "new_booking",
		Title:   title,
		Message: message,
		Data: map[string]any{
			"bookingId": booking.ID,
			"date":      booking.Date,
			"time":      booking.Start,
			"dateTime":  formattedDateTime,
			"timeSlot": map[string]any{
				"id":       slot.ID,
				"start":    slot.Start,
				"end":      slot.End,
				"capacity": remaining,
			},
			"serviceMode":    booking.Mode,
			"actionRequired": false,
			"user": map[string]any{
				"id":           user.ID,
				"username":     user.Username,
				"profileImage": user.ProfileImage,
				"rating":       user.Rating,
				"phoneNumber":  user.PhoneNumber,
			},
		},
		CreatedAt: time.Now(),
		Read:      false,
	}

	activeBooking := models.ActiveBookingDTO{
		BookingID: booking.ID,
		CreatedAt: booking.CreatedAt,
		End:       booking.End,
		Mode:      booking.Mode,
		User: models.UserMinimal{
			ID:           user.ID,
			Username:     user.Username,
			ProfileImage: user.ProfileImage,
			Rating:       user.Rating,
			PhoneNumber:  user.PhoneNumber,
		},
	}

	if booking.Mode == "in_home" {
		activeBooking.User.Location = user.Location
		notification.Data["user"].(map[string]any)["location"] = user.Location
	}

	now := time.Now()

	// Push to MongoDB
	pushDoc := bson.M{
		"notifications":  notification,
		"activeBookings": activeBooking,
	}
	err = se.ProviderRepo.UpdatePushDocument(provider.ID, pushDoc)
	if err != nil {
		log.Printf("[UpdateProviderWithBookingNotification] Failed to update provider %s: %v", provider.ID, err)
		return false
	}

	setDoc := bson.M{
		"updatedAt": now,
	}
	if err := se.ProviderRepo.UpdateSetDocument(provider.ID, setDoc); err != nil {
		return false
	}

	userDetails := map[string]string{
		"userId":      user.ID,
		"username":    user.Username,
		"phoneNumber": user.PhoneNumber,
		"rating":      fmt.Sprintf("%d", user.Rating),
	}

	if booking.Mode == "in_home" && len(user.Location.Coordinates) == 2 {
		userDetails["locationLongitude"] = fmt.Sprintf("%f", user.Location.Coordinates[0])
		userDetails["locationLatitude"] = fmt.Sprintf("%f", user.Location.Coordinates[1])
	}

	go func() {
		notificationData := map[string]string{
			"type":           "new_booking",
			"bookingId":      booking.ID,
			"date":           booking.Date,
			"time":           fmt.Sprintf("%d", booking.Start),
			"dateTime":       formattedDateTime,
			"timeSlotId":     slot.ID,
			"remainingSpots": fmt.Sprintf("%d", remaining),
			"serviceMode":    booking.Mode,
		}
		for k, v := range userDetails {
			notificationData[k] = v
		}

		err := se.Notification.SendProviderPushNotification(
			context.Background(),
			provider.ID,
			title,
			message,
			notificationData,
		)
		if err != nil {
			log.Printf("[PushNotification] Failed to send provider %s notification: %v", provider.ID, err)
		}
	}()

	return true
}
