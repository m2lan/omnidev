package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/billing/internal/domain"
)

// InvoiceRepository defines the interface for invoice data access.
type InvoiceRepository interface {
	Create(ctx context.Context, invoice *domain.Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	ListByAccount(ctx context.Context, accountID uuid.UUID, offset, limit int) ([]*domain.Invoice, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, paymentID *string) error
}

type invoiceRepository struct {
	pool *pgxpool.Pool
}

func NewInvoiceRepository(pool *pgxpool.Pool) InvoiceRepository {
	return &invoiceRepository{pool: pool}
}

func (r *invoiceRepository) Create(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		INSERT INTO billing_invoices (id, billing_account_id, invoice_number, period_start, period_end, subtotal, tax, total, currency, status, line_items, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		invoice.ID, invoice.BillingAccountID, invoice.InvoiceNumber,
		invoice.PeriodStart, invoice.PeriodEnd, invoice.Subtotal,
		invoice.Tax, invoice.Total, invoice.Currency, invoice.Status,
		invoice.LineItems, invoice.Metadata,
	).Scan(&invoice.CreatedAt, &invoice.UpdatedAt)
}

func (r *invoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	query := `
		SELECT id, billing_account_id, invoice_number, period_start, period_end,
		       subtotal, tax, total, currency, status, payment_method, payment_id, paid_at,
		       line_items, metadata, created_at, updated_at
		FROM billing_invoices WHERE id = $1`

	inv := &domain.Invoice{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.BillingAccountID, &inv.InvoiceNumber,
		&inv.PeriodStart, &inv.PeriodEnd, &inv.Subtotal,
		&inv.Tax, &inv.Total, &inv.Currency, &inv.Status,
		&inv.PaymentMethod, &inv.PaymentID, &inv.PaidAt,
		&inv.LineItems, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("invoice not found: %w", err)
	}
	return inv, nil
}

func (r *invoiceRepository) ListByAccount(ctx context.Context, accountID uuid.UUID, offset, limit int) ([]*domain.Invoice, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM billing_invoices WHERE billing_account_id = $1`, accountID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, billing_account_id, invoice_number, period_start, period_end,
		       subtotal, tax, total, currency, status, payment_method, payment_id, paid_at,
		       line_items, metadata, created_at, updated_at
		FROM billing_invoices WHERE billing_account_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	invoices := make([]*domain.Invoice, 0)
	for rows.Next() {
		inv := &domain.Invoice{}
		if err := rows.Scan(
			&inv.ID, &inv.BillingAccountID, &inv.InvoiceNumber,
			&inv.PeriodStart, &inv.PeriodEnd, &inv.Subtotal,
			&inv.Tax, &inv.Total, &inv.Currency, &inv.Status,
			&inv.PaymentMethod, &inv.PaymentID, &inv.PaidAt,
			&inv.LineItems, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		invoices = append(invoices, inv)
	}

	return invoices, total, nil
}

func (r *invoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, paymentID *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_invoices SET status = $1, payment_id = $2 WHERE id = $3`,
		status, paymentID, id)
	return err
}
