package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bloomify/database/repository"
	"bloomify/models"
	"bloomify/services"
	"bloomify/utils"
)

var MatchingService services.MatchingService = &services.DefaultMatchingService{
	// Initialize fields as needed, e.g.:
	ProviderRepo: repository.NewGormProviderRepo(),
	CacheClient:  utils.GetCacheClient(),
}
var BookingService services.BookingService = &services.DefaultBookingService{}
var CacheClient = utils.GetCacheClient()

// StartBookingSession creates a new booking session.
func StartBookingSession(c *gin.Context) {
	var input struct {
		ServicePlan models.ServicePlan `json:"servicePlan"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input", "details": err.Error()})
		return
	}

	// Run matching logic to get providers.
	matchedProviders, err := MatchingService.MatchProviders(input.ServicePlan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to match providers: %v", err)})
		return
	}

	// Create a new booking session.
	session := models.BookingSession{
		ServicePlan:      input.ServicePlan,
		MatchedProviders: matchedProviders,
	}
	sessionID := uuid.New().String()
	sessionData, err := json.Marshal(session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal booking session", "details": err.Error()})
		return
	}

	// Cache the session (e.g., for 10 minutes).
	ctx := context.Background()
	if err := CacheClient.Set(ctx, sessionID, sessionData, 10*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cache booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID": sessionID,
		"providers": matchedProviders,
	})
}

// UpdateBookingSession updates the booking session with provider selection and recalculates availability.
func UpdateBookingSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	var input struct {
		BookingRequest models.BookingRequestInput `json:"bookingRequest"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input", "details": err.Error()})
		return
	}

	ctx := context.Background()
	sessionData, err := CacheClient.Get(ctx, sessionID).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking session not found or expired"})
		return
	}

	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse booking session", "details": err.Error()})
		return
	}

	// Override provider selection.
	session.SelectedProvider = input.BookingRequest.ProviderID

	// Build a request for availability check.
	availReq := services.BookingRequest{
		ProviderID:  session.SelectedProvider,
		UserID:      0, // Not needed for checking availability.
		Date:        input.BookingRequest.Date,
		StartMinute: 0, // Not used for availability calculation.
		Duration:    input.BookingRequest.Duration,
		Units:       input.BookingRequest.Units,
	}
	availIntervals, err := BookingService.CheckAvailability(availReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to compute availability: %v", err)})
		return
	}
	session.Availability = availIntervals

	updatedData, err := json.Marshal(session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal updated session", "details": err.Error()})
		return
	}
	if err := CacheClient.Set(ctx, sessionID, updatedData, 10*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID":    sessionID,
		"availability": availIntervals,
		"providers":    session.MatchedProviders,
	})
}

// ConfirmBooking finalizes the booking.
func ConfirmBooking(c *gin.Context) {
	var input struct {
		SessionID      string                     `json:"sessionID"`
		BookingRequest models.BookingRequestInput `json:"bookingRequest"`
		ConfirmedSlot  models.AvailableInterval   `json:"confirmedSlot"`
		UserID         uint                       `json:"userID"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input", "details": err.Error()})
		return
	}

	ctx := context.Background()
	sessionData, err := CacheClient.Get(ctx, input.SessionID).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking session not found or expired"})
		return
	}
	var session models.BookingSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse booking session", "details": err.Error()})
		return
	}

	if session.SelectedProvider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider not selected; cannot confirm booking"})
		return
	}

	// Build the final booking request.
	finalReq := services.BookingRequest{
		ProviderID:  session.SelectedProvider,
		UserID:      input.UserID,
		Date:        input.BookingRequest.Date,
		StartMinute: input.ConfirmedSlot.Start,
		Duration:    input.BookingRequest.Duration,
		Units:       input.BookingRequest.Units,
	}
	confirmedBooking, err := BookingService.BookSlot(finalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("booking confirmation failed: %v", err)})
		return
	}

	// Clear the session from cache.
	CacheClient.Del(ctx, input.SessionID)

	c.JSON(http.StatusOK, gin.H{
		"booking": confirmedBooking,
	})
}
