// Package domain defines the core business entities for the Billing Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanType represents subscription plan types.
type PlanType string

const (
	PlanFree       PlanType = "free"
	PlanPro        PlanType = "pro"
	PlanTeam       PlanType = "team"
	PlanEnterprise PlanType = "enterprise"
)

// InvoiceStatus represents invoice status.
type InvoiceStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "draft"
	InvoiceStatusPending InvoiceStatus = "pending"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusFailed  InvoiceStatus = "failed"
	InvoiceStatusRefunded InvoiceStatus = "refunded"
)

// BillingAccount represents a billing account.
type BillingAccount struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	UserID          *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	OrgID           *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Plan            PlanType               `json:"plan" db:"plan"`
	Balance         float64                `json:"balance" db:"balance"`
	Credit          float64                `json:"credit" db:"credit"`
	MonthlyBudget   *float64               `json:"monthly_budget,omitempty" db:"monthly_budget"`
	AlertThreshold  float64                `json:"alert_threshold" db:"alert_threshold"`
	StripeCustomerID *string               `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	PaymentMethod   map[string]interface{} `json:"payment_method,omitempty" db:"payment_method"`
	Status          string                 `json:"status" db:"status"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// UsageRecord represents a usage record.
type UsageRecord struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	BillingAccountID uuid.UUID  `json:"billing_account_id" db:"billing_account_id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	Service          string     `json:"service" db:"service"`
	ModelID          *uuid.UUID `json:"model_id,omitempty" db:"model_id"`
	InputTokens      int64      `json:"input_tokens" db:"input_tokens"`
	OutputTokens     int64      `json:"output_tokens" db:"output_tokens"`
	Cost             float64    `json:"cost" db:"cost"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	RecordedAt       time.Time  `json:"recorded_at" db:"recorded_at"`
}

// Invoice represents a billing invoice.
type Invoice struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	BillingAccountID uuid.UUID             `json:"billing_account_id" db:"billing_account_id"`
	InvoiceNumber   string                 `json:"invoice_number" db:"invoice_number"`
	PeriodStart     time.Time              `json:"period_start" db:"period_start"`
	PeriodEnd       time.Time              `json:"period_end" db:"period_end"`
	Subtotal        float64                `json:"subtotal" db:"subtotal"`
	Tax             float64                `json:"tax" db:"tax"`
	Total           float64                `json:"total" db:"total"`
	Currency        string                 `json:"currency" db:"currency"`
	Status          InvoiceStatus          `json:"status" db:"status"`
	PaymentMethod   *string                `json:"payment_method,omitempty" db:"payment_method"`
	PaymentID       *string                `json:"payment_id,omitempty" db:"payment_id"`
	PaidAt          *time.Time             `json:"paid_at,omitempty" db:"paid_at"`
	LineItems       []LineItem             `json:"line_items" db:"line_items"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// LineItem represents an invoice line item.
type LineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// UsageSummary represents aggregated usage.
type UsageSummary struct {
	TotalInputTokens  int64              `json:"total_input_tokens"`
	TotalOutputTokens int64              `json:"total_output_tokens"`
	TotalCost         float64            `json:"total_cost"`
	ByService         map[string]float64 `json:"by_service"`
	ByModel           map[string]float64 `json:"by_model"`
	Period            string             `json:"period"`
}
