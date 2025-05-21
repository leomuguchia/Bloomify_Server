package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/services/user"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"go.uber.org/zap"
)

type PaymentHandler interface {
	ProcessPayment(ctx context.Context, req models.PaymentRequest) (*models.Invoice, error)
}

type UnifiedPaymentHandler struct {
	logger      *zap.Logger
	userService user.UserService
}

func NewPaymentHandler(
	logger *zap.Logger,
	userService user.UserService,
) *UnifiedPaymentHandler {
	return &UnifiedPaymentHandler{
		logger:      logger,
		userService: userService,
	}
}

func (h *UnifiedPaymentHandler) ProcessPayment(
	ctx context.Context,
	req models.PaymentRequest,
) (*models.Invoice, error) {

	if err := validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid payment request: %w", err)
	}

	switch req.Method {
	case "cash":
		return h.processCashPayment(ctx, req)
	case "card":
		switch req.Action {
		case "authorize":
			return h.authorizeCardPayment(ctx, req)
		case "capture":
			return h.captureCardPayment(ctx, req.PaymentIntentID, req)
		case "cancel":
			return nil, h.cancelCardPayment(ctx, req.PaymentIntentID)
		default:
			return nil, fmt.Errorf("unsupported card action: %s", req.Action)
		}
	default:
		return nil, fmt.Errorf("unsupported payment method: %s", req.Method)
	}
}

// ---------------------------------------------------------------------
// VALIDATION
// ---------------------------------------------------------------------

func validateRequest(req models.PaymentRequest) error {
	if req.Amount <= 0 {
		return errors.New("invalid payment amount")
	}
	if req.UserID == "" {
		return errors.New("missing user ID")
	}
	switch req.Method {
	case "cash":
		return nil
	case "card":
		if req.PaymentIntentID == "" {
			return errors.New("missing PaymentIntent ID for card payment")
		}
		if req.Action == "" {
			return errors.New("missing Action for card payment (authorize|capture|cancel)")
		}
		return nil
	default:
		return fmt.Errorf("unsupported payment method: %s", req.Method)
	}
}

// ---------------------------------------------------------------------
// CASH PAYMENT
// ---------------------------------------------------------------------

func (h *UnifiedPaymentHandler) processCashPayment(
	ctx context.Context,
	req models.PaymentRequest,
) (*models.Invoice, error) {

	inv := &models.Invoice{
		InvoiceID: uuid.New().String(),
		UserID:    req.UserID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    "cash",
		Status:    "cash_pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.logger.Info("Cash payment recorded", zap.String("invoiceID", inv.InvoiceID), zap.String("userID", req.UserID))

	if err := h.updateUserAfterPayment(ctx, req.UserID, inv); err != nil {
		h.logger.Error("Failed to update user with payment notification", zap.Error(err))
	}

	return inv, nil
}

func (h *UnifiedPaymentHandler) authorizeCardPayment(
	ctx context.Context,
	req models.PaymentRequest,
) (*models.Invoice, error) {

	intent, err := paymentintent.Get(req.PaymentIntentID, nil)
	if err != nil {
		h.logger.Error("Stripe: unable to fetch PaymentIntent", zap.Error(err))
		return nil, fmt.Errorf("stripe verification failed: %w", err)
	}

	if intent.Status != stripe.PaymentIntentStatusRequiresCapture {
		h.logger.Warn("Stripe intent not in requires_capture", zap.String("status", string(intent.Status)))
		return nil, fmt.Errorf("payment not authorized, status: %s", intent.Status)
	}

	inv := &models.Invoice{
		InvoiceID: uuid.New().String(),
		UserID:    req.UserID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    "card",
		PaymentID: intent.ID,
		Status:    "authorized",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.logger.Info("Payment authorized", zap.String("invoiceID", inv.InvoiceID))

	return inv, nil
}

func (h *UnifiedPaymentHandler) captureCardPayment(
	ctx context.Context,
	intentID string,
	req models.PaymentRequest,
) (*models.Invoice, error) {

	pi, err := paymentintent.Capture(intentID, nil)
	if err != nil {
		h.logger.Error("Stripe capture failed", zap.Error(err))
		return nil, fmt.Errorf("stripe capture failed: %w", err)
	}

	inv := &models.Invoice{
		InvoiceID: uuid.New().String(),
		UserID:    req.UserID,
		Amount:    float64(pi.Amount) / 100.0,
		Currency:  string(pi.Currency),
		Method:    "card",
		PaymentID: pi.ID,
		Status:    "paid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.logger.Info("Payment captured", zap.String("invoiceID", inv.InvoiceID))

	if err := h.updateUserAfterPayment(ctx, req.UserID, inv); err != nil {
		h.logger.Error("Failed to  update user with payment notification", zap.Error(err))
	}

	return inv, nil
}

func (h *UnifiedPaymentHandler) cancelCardPayment(
	ctx context.Context,
	intentID string,
) error {

	_, err := paymentintent.Cancel(intentID, nil)
	if err != nil {
		h.logger.Error("Stripe cancel failed", zap.Error(err))
		return fmt.Errorf("stripe cancel failed: %w", err)
	}

	h.logger.Info("PaymentIntent canceled", zap.String("intentID", intentID))
	return nil
}

func (h *UnifiedPaymentHandler) updateUserAfterPayment(
	ctx context.Context,
	userID string,
	inv *models.Invoice,
) error {
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	notification := models.Notification{
		ID:      uuid.New().String(),
		Type:    "payment_confirmation",
		Message: fmt.Sprintf("Payment of %s %.2f via %s was %s.", inv.Currency, inv.Amount, inv.Method, inv.Status),
		Data: map[string]any{
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
