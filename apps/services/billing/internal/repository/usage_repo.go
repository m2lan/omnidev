package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/billing/internal/domain"
)

// UsageRepository defines the interface for usage data access.
type UsageRepository interface {
	Record(ctx context.Context, record *domain.UsageRecord) error
	GetSummary(ctx context.Context, accountID uuid.UUID, start, end time.Time) (*domain.UsageSummary, error)
	GetByUser(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.UsageRecord, error)
}

type usageRepository struct {
	pool *pgxpool.Pool
}

func NewUsageRepository(pool *pgxpool.Pool) UsageRepository {
	return &usageRepository{pool: pool}
}

func (r *usageRepository) Record(ctx context.Context, record *domain.UsageRecord) error {
	query := `
		INSERT INTO usage_records (id, billing_account_id, user_id, service, model_id, input_tokens, output_tokens, cost, metadata, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		record.ID, record.BillingAccountID, record.UserID,
		record.Service, record.ModelID, record.InputTokens,
		record.OutputTokens, record.Cost, record.Metadata, record.RecordedAt,
	)
	return err
}

func (r *usageRepository) GetSummary(ctx context.Context, accountID uuid.UUID, start, end time.Time) (*domain.UsageSummary, error) {
	query := `
		SELECT COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cost), 0)
		FROM usage_records
		WHERE billing_account_id = $1 AND recorded_at BETWEEN $2 AND $3`

	summary := &domain.UsageSummary{
		ByService: make(map[string]float64),
		ByModel:   make(map[string]float64),
		Period:    fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
	}

	err := r.pool.QueryRow(ctx, query, accountID, start, end).Scan(
		&summary.TotalInputTokens, &summary.TotalOutputTokens, &summary.TotalCost,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}

	// Get by service
	svcQuery := `
		SELECT service, COALESCE(SUM(cost), 0)
		FROM usage_records
		WHERE billing_account_id = $1 AND recorded_at BETWEEN $2 AND $3
		GROUP BY service`

	rows, err := r.pool.Query(ctx, svcQuery, accountID, start, end)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var service string
			var cost float64
			if err := rows.Scan(&service, &cost); err == nil {
				summary.ByService[service] = cost
			}
		}
	}

	return summary, nil
}

func (r *usageRepository) GetByUser(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.UsageRecord, error) {
	query := `
		SELECT id, billing_account_id, user_id, service, model_id, input_tokens, output_tokens, cost, metadata, recorded_at
		FROM usage_records
		WHERE user_id = $1 AND recorded_at BETWEEN $2 AND $3
		ORDER BY recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]domain.UsageRecord, 0)
	for rows.Next() {
		rec := domain.UsageRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.BillingAccountID, &rec.UserID,
			&rec.Service, &rec.ModelID, &rec.InputTokens,
			&rec.OutputTokens, &rec.Cost, &rec.Metadata, &rec.RecordedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, nil
}
