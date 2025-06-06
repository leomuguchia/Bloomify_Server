// File: service/ai/default_service.go
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"bloomify/models"
	"bloomify/services/booking"
)

type DefaultAIService struct {
	gemini   *GeminiClient
	ctxStore *RedisContextStore
	bookSvc  booking.BookingSessionService
}

func NewDefaultAIService(
	makerSuiteKey string,
	ctxStore *RedisContextStore,
	bookSvc booking.BookingSessionService,
) *DefaultAIService {
	return &DefaultAIService{
		gemini:   NewGeminiClient(makerSuiteKey),
		ctxStore: ctxStore,
		bookSvc:  bookSvc,
	}
}

// ProcessUserInput handles every user turn.
func (s *DefaultAIService) ProcessUserInput(req models.AIRequest) (*models.AIResponse, error) {
	ctx := context.Background()

	// 1) Load existing context (BookingStep >=1 means mid-booking)
	aiCtx, err := s.ctxStore.Get(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("load context: %w", err)
	}

	// 2) If mid-booking, continue the 3-step flow
	if aiCtx.BookingStep > 0 {
		return s.handleBookingFlow(ctx, req, aiCtx)
	}

	// 3) Otherwise extract intent + serviceType
	intent, svcType, err := s.getIntentAndService(ctx, req.Text)
	if err != nil {
		return nil, fmt.Errorf("intent extraction: %w", err)
	}

	// 4) Save serviceType and reset booking state
	aiCtx.ServiceType = svcType
	aiCtx.BookingStep = 0
	aiCtx.BookingSessID = ""
	if err := s.ctxStore.Set(ctx, req.UserID, aiCtx); err != nil {
		return nil, fmt.Errorf("save context: %w", err)
	}

	// 5) Branch by intent
	switch intent {
	case "chat", "recommend":
		return s.handleChatOrRecommend(ctx, req, intent, svcType)

	case "book":
		// User clicked “Book X” → start booking flow
		aiCtx.BookingStep = 1
		if err := s.ctxStore.Set(ctx, req.UserID, aiCtx); err != nil {
			return nil, fmt.Errorf("save context: %w", err)
		}
		return &models.AIResponse{
			Intent:       "book",
			ServiceType:  svcType,
			ResponseText: fmt.Sprintf("Sure—finding providers for %s now. Ready?", svcType),
			Actions: []models.AIAction{
				{Label: "Yes, find providers", Type: "book", ServiceID: svcType},
				{Label: "Not now", Type: "chat"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported intent: %q", intent)
	}
}

// handleChatOrRecommend generates natural replies with no actions.
func (s *DefaultAIService) handleChatOrRecommend(
	ctx context.Context,
	req models.AIRequest,
	intent, svcType string,
) (*models.AIResponse, error) {
	// We only use service names in the system prompt—no actions emitted
	services, err := s.bookSvc.GetAvailableServices("")
	if err != nil {
		return nil, fmt.Errorf("load services: %w", err)
	}
	names := make([]string, len(services))
	for i, svc := range services {
		names[i] = svc.ID
	}

	var prompt string
	if intent == "chat" {
		prompt = fmt.Sprintf(
			`You are Bloomify Assistant, focused on these services: %v.
User said: "%s"
Greet them and ask how you can help.`,
			names, req.Text,
		)
	} else {
		prompt = fmt.Sprintf(
			`User needs help: "%s".
Recommend 2–3 services from this list: %v, with brief descriptions.`,
			req.Text, names,
		)
	}

	text, err := s.callGemini(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("chat/recommend GPT: %w", err)
	}

	return &models.AIResponse{
		Intent:       intent,
		ServiceType:  svcType,
		ResponseText: text,
		Actions:      nil,
	}, nil
}

// handleBookingFlow runs your 3-step booking logic.
func (s *DefaultAIService) handleBookingFlow(
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

func (s *DefaultAIService) getIntentAndService(
	ctx context.Context,
	text string,
) (string, string, error) {
	services, err := s.bookSvc.GetAvailableServices("")
	if err != nil {
		return "", "", err
	}
	names := make([]string, len(services))
	for i, svc := range services {
		names[i] = svc.ID
	}

	prompt := fmt.Sprintf(`You are Bloomify’s NLP component.
Available services: %v.

Given the user’s message, respond ONLY in raw JSON:
{"intent":"chat|recommend|book","serviceType":"<one_of_the_above_or_empty>"}

User message: "%s"`, names, text)

	respText, err := s.callGemini(ctx, prompt)
	if err != nil {
		return "", "", err
	}

	var slot struct {
		Intent      string `json:"intent"`
		ServiceType string `json:"serviceType"`
	}
	if err := json.Unmarshal([]byte(respText), &slot); err != nil {
		return "", "", fmt.Errorf("gemini intent JSON parse error: %w\nRaw: %s", err, respText)
	}
	return slot.Intent, slot.ServiceType, nil
}

func (s *DefaultAIService) callGemini(ctx context.Context, prompt string) (string, error) {
	return s.gemini.GenerateContent(ctx, prompt)
}
