// Package service contains the business logic for the Billing Service.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/billing/internal/domain"
	"github.com/omnidev/services/billing/internal/payment"
	"github.com/omnidev/services/billing/internal/repository"
)

// BillingService handles billing operations.
type BillingService struct {
	accountRepo repository.AccountRepository
	usageRepo   repository.UsageRepository
	invoiceRepo repository.InvoiceRepository
	stripe      *payment.StripeProvider
	wechat      *payment.WechatProvider
	alipay      *payment.AlipayProvider
}

// NewBillingService creates a new billing service.
func NewBillingService(
	accountRepo repository.AccountRepository,
	usageRepo repository.UsageRepository,
	invoiceRepo repository.InvoiceRepository,
	stripe *payment.StripeProvider,
	wechat *payment.WechatProvider,
	alipay *payment.AlipayProvider,
) *BillingService {
	return &BillingService{
		accountRepo: accountRepo,
		usageRepo:   usageRepo,
		invoiceRepo: invoiceRepo,
		stripe:      stripe,
		wechat:      wechat,
		alipay:      alipay,
	}
}

// GetOrCreateAccount gets or creates a billing account for a user.
func (s *BillingService) GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*domain.BillingAccount, error) {
	account, err := s.accountRepo.GetByUserID(ctx, userID)
	if err == nil {
		return account, nil
	}

	// Create new account
	account = &domain.BillingAccount{
		ID:             uuid.New(),
		UserID:         &userID,
		Plan:           domain.PlanFree,
		Balance:        0,
		Credit:         10.0, // $10 free credit
		AlertThreshold: 80,
		Status:         "active",
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, appErr.Wrap(err, "failed to create billing account")
	}

	return account, nil
}

// GetAccount returns a billing account by user ID.
func (s *BillingService) GetAccount(ctx context.Context, userID uuid.UUID) (*domain.BillingAccount, error) {
	return s.GetOrCreateAccount(ctx, userID)
}

// RecordUsageInput defines the input for recording usage.
type RecordUsageInput struct {
	Service      string     `json:"service" validate:"required"`
	ModelID      *uuid.UUID `json:"model_id,omitempty"`
	InputTokens  int64      `json:"input_tokens"`
	OutputTokens int64      `json:"output_tokens"`
	Cost         float64    `json:"cost"`
}

// RecordUsage records a usage event.
func (s *BillingService) RecordUsage(ctx context.Context, userID uuid.UUID, input *RecordUsageInput) (*domain.UsageRecord, error) {
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	record := &domain.UsageRecord{
		ID:               uuid.New(),
		BillingAccountID: account.ID,
		UserID:           userID,
		Service:          input.Service,
		ModelID:          input.ModelID,
		InputTokens:      input.InputTokens,
		OutputTokens:     input.OutputTokens,
		Cost:             input.Cost,
		RecordedAt:       time.Now(),
		Metadata:         map[string]interface{}{},
	}

	if err := s.usageRepo.Record(ctx, record); err != nil {
		return nil, appErr.Wrap(err, "failed to record usage")
	}

	// Deduct from balance
	if input.Cost > 0 {
		_ = s.accountRepo.UpdateBalance(ctx, account.ID, -input.Cost)
	}

	// Check budget alert
	if account.MonthlyBudget != nil {
		summary, err := s.usageRepo.GetSummary(ctx, account.ID,
			time.Now().AddDate(0, 0, -30), time.Now())
		if err == nil && summary.TotalCost >= *account.MonthlyBudget*(account.AlertThreshold/100) {
			logger.Log.Warn("Budget alert threshold reached",
				zap.String("user_id", userID.String()),
				zap.Float64("cost", summary.TotalCost),
				zap.Float64("budget", *account.MonthlyBudget),
			)
		}
	}

	return record, nil
}

// GetUsage returns usage summary for a user.
func (s *BillingService) GetUsage(ctx context.Context, userID uuid.UUID, days int) (*domain.UsageSummary, error) {
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	start := time.Now().AddDate(0, 0, -days)
	end := time.Now()

	return s.usageRepo.GetSummary(ctx, account.ID, start, end)
}

// ListInvoices returns invoices for a user.
func (s *BillingService) ListInvoices(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Invoice, int, error) {
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.invoiceRepo.ListByAccount(ctx, account.ID, offset, pageSize)
}

// GetInvoice returns an invoice by ID.
func (s *BillingService) GetInvoice(ctx context.Context, userID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, appErr.NotFound("invoice")
	}

	// Verify ownership
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}
	if inv.BillingAccountID != account.ID {
		return nil, appErr.ErrForbidden
	}

	return inv, nil
}

// SubscribeInput defines the input for subscribing to a plan.
type SubscribeInput struct {
	Plan           string `json:"plan" validate:"required,oneof=pro team enterprise"`
	PaymentMethod  string `json:"payment_method" validate:"required,oneof=stripe wechat alipay"`
}

// Subscribe subscribes a user to a plan.
func (s *BillingService) Subscribe(ctx context.Context, userID uuid.UUID, input *SubscribeInput) error {
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return err
	}

	plan := domain.PlanType(input.Plan)

	// Update plan
	if err := s.accountRepo.UpdatePlan(ctx, account.ID, plan); err != nil {
		return appErr.Wrap(err, "failed to update plan")
	}

	logger.Log.Info("User subscribed",
		zap.String("user_id", userID.String()),
		zap.String("plan", input.Plan),
	)

	return nil
}

// AddPaymentMethodInput defines the input for adding a payment method.
type AddPaymentMethodInput struct {
	Provider string `json:"provider" validate:"required,oneof=stripe wechat alipay"`
	Token    string `json:"token" validate:"required"`
}

// AddPaymentMethod adds a payment method to a billing account.
func (s *BillingService) AddPaymentMethod(ctx context.Context, userID uuid.UUID, input *AddPaymentMethodInput) error {
	account, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return err
	}

	var provider payment.Provider
	switch input.Provider {
	case "stripe":
		provider = s.stripe
	case "wechat":
		provider = s.wechat
	case "alipay":
		provider = s.alipay
	default:
		return appErr.Validation("unsupported payment provider")
	}

	customerID, err := provider.CreateCustomer(userID.String(), "")
	if err != nil {
		return appErr.Wrap(err, "failed to create customer")
	}

	logger.Log.Info("Payment method added",
		zap.String("user_id", userID.String()),
		zap.String("provider", input.Provider),
		zap.String("customer_id", customerID),
	)

	_ = account
	return nil
}
