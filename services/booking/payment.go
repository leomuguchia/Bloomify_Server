package booking

import (
	"bloomify/models"
	"bloomify/services/user"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// --- Interfaces ---
type PaymentHandler interface {
	ProcessPayment(ctx context.Context, req models.PaymentRequest) (*models.Invoice, error)
}

// --- PaymentHandler Implementation ---
type UnifiedPaymentHandler struct {
	logger      *zap.Logger
	userService user.UserService
	// mu          sync.Mutex
}

// --- NewPaymentHandler Constructor ---
func NewPaymentHandler(logger *zap.Logger, userService user.UserService) *UnifiedPaymentHandler {
	return &UnifiedPaymentHandler{
		logger:      logger,
		userService: userService,
	}
}

// --- ProcessPayment Entry Point ---
func (h *UnifiedPaymentHandler) ProcessPayment(ctx context.Context, req models.PaymentRequest) (*models.Invoice, error) {
	if err := validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid payment request: %w", err)
	}

	inv := &models.Invoice{
		InvoiceID: uuid.New().String(),
		UserID:    req.UserID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    req.Method,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	switch req.Method {
	case "card":
		return h.processCardPayment(ctx, req, inv)
	case "cash":
		return h.processCashPayment(ctx, req, inv)
	default:
		return nil, fmt.Errorf("unsupported payment method: %s", req.Method)
	}
}

// --- Card Payment Processing ---
func (h *UnifiedPaymentHandler) processCardPayment(ctx context.Context, req models.PaymentRequest, inv *models.Invoice) (*models.Invoice, error) {
	time.Sleep(1 * time.Second) // Simulate card payment processing

	inv.PaymentID = "pi_" + uuid.New().String() // Simulated PaymentID for the card
	inv.Status = "paid"
	inv.UpdatedAt = time.Now()

	if err := h.updateUserAfterPayment(ctx, req, inv); err != nil {
		h.logger.Error("user update failed", zap.Error(err))
	}

	h.logger.Info("Card payment successful", zap.String("invoice", inv.InvoiceID))
	return inv, nil
}

// --- Cash Payment Processing ---
func (h *UnifiedPaymentHandler) processCashPayment(ctx context.Context, req models.PaymentRequest, inv *models.Invoice) (*models.Invoice, error) {
	time.Sleep(500 * time.Millisecond) // Simulate cash entry delay

	// Cash payment remains "pending" status
	inv.UpdatedAt = time.Now()

	if err := h.updateUserAfterPayment(ctx, req, inv); err != nil {
		h.logger.Error("user update failed", zap.Error(err))
	}

	h.logger.Info("Cash payment recorded", zap.String("invoice", inv.InvoiceID))
	return inv, nil
}

// --- User Update ---
func (h *UnifiedPaymentHandler) updateUserAfterPayment(ctx context.Context, req models.PaymentRequest, inv *models.Invoice) error {
	user, err := h.userService.GetUserByID(req.UserID)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    "payment_confirmation",
		Message: fmt.Sprintf("Payment of %s %.2f via %s was %s.", inv.Currency, inv.Amount, inv.Method, inv.Status),
		Data: map[string]interface{}{
			"invoiceId": inv.InvoiceID,
			"amount":    inv.Amount,
			"method":    inv.Method,
			"status":    inv.Status,
		},
		CreatedAt: time.Now(),
		Read:      false,
	}

	user.Notifications = append(user.Notifications, notification)
	user.UpdatedAt = time.Now()

	_, err = h.userService.UpdateUser(*user)
	return err
}

// --- Validator ---
func validateRequest(req models.PaymentRequest) error {
	if req.Amount <= 0 {
		return errors.New("invalid payment amount")
	}
	if req.UserID == "" {
		return errors.New("missing user ID")
	}
	if req.Method != "card" && req.Method != "cash" {
		return errors.New("unsupported method")
	}
	return nil
}
