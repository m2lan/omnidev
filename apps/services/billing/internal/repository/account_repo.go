// Package repository provides data access implementations for the Billing Service.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/billing/internal/domain"
)

// AccountRepository defines the interface for billing account data access.
type AccountRepository interface {
	Create(ctx context.Context, account *domain.BillingAccount) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.BillingAccount, error)
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*domain.BillingAccount, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BillingAccount, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, amount float64) error
	UpdatePlan(ctx context.Context, id uuid.UUID, plan domain.PlanType) error
}

type accountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) AccountRepository {
	return &accountRepository{pool: pool}
}

func (r *accountRepository) Create(ctx context.Context, account *domain.BillingAccount) error {
	query := `
		INSERT INTO billing_accounts (id, user_id, org_id, plan, balance, credit, monthly_budget, alert_threshold, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		account.ID, account.UserID, account.OrgID, account.Plan,
		account.Balance, account.Credit, account.MonthlyBudget,
		account.AlertThreshold, account.Status,
	).Scan(&account.CreatedAt, &account.UpdatedAt)
}

func (r *accountRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.BillingAccount, error) {
	query := `
		SELECT id, user_id, org_id, plan, balance, credit, monthly_budget, alert_threshold,
		       stripe_customer_id, payment_method, status, created_at, updated_at
		FROM billing_accounts WHERE user_id = $1`

	account := &domain.BillingAccount{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&account.ID, &account.UserID, &account.OrgID, &account.Plan,
		&account.Balance, &account.Credit, &account.MonthlyBudget,
		&account.AlertThreshold, &account.StripeCustomerID,
		&account.PaymentMethod, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("billing account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get billing account: %w", err)
	}
	return account, nil
}

func (r *accountRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*domain.BillingAccount, error) {
	query := `
		SELECT id, user_id, org_id, plan, balance, credit, monthly_budget, alert_threshold,
		       stripe_customer_id, payment_method, status, created_at, updated_at
		FROM billing_accounts WHERE org_id = $1`

	account := &domain.BillingAccount{}
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&account.ID, &account.UserID, &account.OrgID, &account.Plan,
		&account.Balance, &account.Credit, &account.MonthlyBudget,
		&account.AlertThreshold, &account.StripeCustomerID,
		&account.PaymentMethod, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("billing account not found")
	}
	return account, err
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BillingAccount, error) {
	query := `
		SELECT id, user_id, org_id, plan, balance, credit, monthly_budget, alert_threshold,
		       stripe_customer_id, payment_method, status, created_at, updated_at
		FROM billing_accounts WHERE id = $1`

	account := &domain.BillingAccount{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&account.ID, &account.UserID, &account.OrgID, &account.Plan,
		&account.Balance, &account.Credit, &account.MonthlyBudget,
		&account.AlertThreshold, &account.StripeCustomerID,
		&account.PaymentMethod, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("billing account not found")
	}
	return account, err
}

func (r *accountRepository) UpdateBalance(ctx context.Context, id uuid.UUID, amount float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance + $1 WHERE id = $2`,
		amount, id)
	return err
}

func (r *accountRepository) UpdatePlan(ctx context.Context, id uuid.UUID, plan domain.PlanType) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_accounts SET plan = $1 WHERE id = $2`,
		plan, id)
	return err
}
