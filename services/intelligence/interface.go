// File: service/ai/local_service.go
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"bloomify/models"
	"bloomify/services/booking"
)

type LocalAIService struct {
	ctxStore *RedisContextStore
	bookSvc  booking.BookingSessionService
}

func NewLocalAIService(
	ctxStore *RedisContextStore,
	bookSvc booking.BookingSessionService,
) *LocalAIService {
	return &LocalAIService{
		ctxStore: ctxStore,
		bookSvc:  bookSvc,
	}
}

func (s *LocalAIService) ProcessUserInput(req models.AIRequest) (*models.AIResponse, error) {
	ctx := context.Background()

	// 1) Load existing context
	aiCtx, err := s.ctxStore.Get(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("load context: %w", err)
	}

	// 2) If mid-booking, continue the flow
	if aiCtx.BookingStep > 0 {
		return s.handleBookingFlow(ctx, req, aiCtx)
	}

	// 3) Extract intent using local processing
	intent, svcType := s.getIntentAndService(req.Text)

	// 4) Save context
	aiCtx.ServiceType = svcType
	aiCtx.BookingStep = 0
	aiCtx.BookingSessID = ""
	if err := s.ctxStore.Set(ctx, req.UserID, aiCtx); err != nil {
		return nil, fmt.Errorf("save context: %w", err)
	}

	// 5) Handle intent
	switch intent {
	case "chat":
		return s.handleChat(req, svcType)
	case "recommend":
		return s.handleRecommend(req, svcType)
	case "book":
		aiCtx.BookingStep = 1
		if err := s.ctxStore.Set(ctx, req.UserID, aiCtx); err != nil {
			return nil, fmt.Errorf("save context: %w", err)
		}
		return &models.AIResponse{
			Intent:       "book",
			ServiceType:  svcType,
			ResponseText: fmt.Sprintf("Let's book %s for you. Ready to proceed?", svcType),
			Actions: []models.AIAction{
				{Label: "Yes, continue", Type: "book", ServiceID: svcType},
				{Label: "Not now", Type: "chat"},
			},
		}, nil
	default:
		return s.handleChat(req, "")
	}
}

func (s *LocalAIService) getIntentAndService(text string) (string, string) {
	lowerText := strings.ToLower(text)

	// Simple keyword matching for intent
	var intent string
	switch {
	case strings.Contains(lowerText, "book") || strings.Contains(lowerText, "schedule"):
		intent = "book"
	case strings.Contains(lowerText, "recommend") || strings.Contains(lowerText, "suggest"):
		intent = "recommend"
	default:
		intent = "chat"
	}

	// Get available services for service type matching
	services, err := s.bookSvc.GetAvailableServices("")
	if err != nil {
		return intent, ""
	}

	// Match service type by name
	for _, svc := range services {
		if strings.Contains(lowerText, strings.ToLower(svc.ID)) {
			return intent, svc.ID
		}
	}

	return intent, ""
}

func (s *LocalAIService) handleChat(req models.AIRequest, svcType string) (*models.AIResponse, error) {
	responses := []string{
		"How can I help you today?",
		"I'm here to assist you. What do you need?",
		"Thanks for your message. How can I help?",
	}

	return &models.AIResponse{
		Intent:       "chat",
		ServiceType:  svcType,
		ResponseText: responses[rand.Intn(len(responses))],
	}, nil
}

func (s *LocalAIService) handleRecommend(req models.AIRequest, svcType string) (*models.AIResponse, error) {
	services, err := s.bookSvc.GetAvailableServices("")
	if err != nil {
		return nil, fmt.Errorf("load services: %w", err)
	}

	// If specific service type was mentioned, recommend similar services
	var recommendations []string
	if svcType != "" {
		for _, svc := range services {
			if svc.ID != svcType {
				recommendations = append(recommendations, svc.ID)
				if len(recommendations) >= 3 {
					break
				}
			}
		}
	} else {
		// Recommend random services
		for i := 0; i < 3 && i < len(services); i++ {
			recommendations = append(recommendations, services[i].ID)
		}
	}

	responseText := "You might be interested in these services: " + strings.Join(recommendations, ", ")
	return &models.AIResponse{
		Intent:       "recommend",
		ServiceType:  svcType,
		ResponseText: responseText,
	}, nil
}

// handleBookingFlow runs your 3-step booking logic.
func (s *LocalAIService) handleBookingFlow(
	ctx context.Context,
	req models.AIRequest,
	aiCtx *models.AIContext,
) (*models.AIResponse, error) {
	var respText string
	var actions []models.AIAction

	switch aiCtx.BookingStep {
	case 1:
		// Build minimal ServicePlan
		services, _ := s.bookSvc.GetAvailableServices("")
		var unitType string
		for _, svc := range services {
			if svc.ID == aiCtx.ServiceType {
				unitType = svc.UnitType
				break
			}
		}
		plan := models.ServicePlan{
			ServiceType: aiCtx.ServiceType,
			BookingFor:  "", // free-form, not used by booking engine
			Priority:    false,
			Mode:        models.ModeInHome,
			LocationGeo: models.GeoPoint{}, // assume set elsewhere or default
			Date:        "",                // assume no fixed date yet
			Units:       1,
			UnitType:    unitType,
		}

		sessID, providers, err := s.bookSvc.InitiateSession(plan, req.UserID, "", "")
		if err != nil {
			return nil, err
		}
		aiCtx.BookingSessID = sessID
		aiCtx.BookingStep = 2
		_ = s.ctxStore.Set(ctx, req.UserID, aiCtx)

		respText = "Here are providers near you. Which one would you like?"
		for _, p := range providers {
			actions = append(actions, models.AIAction{
				Label:      p.Profile.ProviderName,
				Type:       "select_provider",
				ProviderID: p.ID,
			})
		}

	case 2:
		// User chose a provider
		providerID := req.Text
		session, err := s.bookSvc.UpdateSession(aiCtx.BookingSessID, providerID, 0)
		if err != nil {
			return nil, err
		}
		aiCtx.BookingStep = 3
		_ = s.ctxStore.Set(ctx, req.UserID, aiCtx)

		respText = "These slots are available. Which date/time works for you?"
		for _, slot := range session.Availability {
			actions = append(actions, models.AIAction{
				Label:       slot.Catalogue.Service.ID,
				Type:        "select_slot",
				Description: slot.Message,
			})
		}

	case 3:
		// User provided slot JSON
		var slotResp models.AvailableSlotResponse
		if err := json.Unmarshal([]byte(req.Text), &slotResp); err != nil {
			return nil, err
		}
		booking, err := s.bookSvc.ConfirmBooking(aiCtx.BookingSessID, slotResp)
		if err != nil {
			return nil, err
		}
		respText = fmt.Sprintf("✅ Booking confirmed! ID %s on %s", booking.ID, booking.Date)
		aiCtx.BookingStep = 0
		_ = s.ctxStore.Clear(ctx, req.UserID)
	}

	return &models.AIResponse{
		Intent:       "book",
		ServiceType:  aiCtx.ServiceType,
		ResponseText: respText,
		Actions:      actions,
	}, nil
}
