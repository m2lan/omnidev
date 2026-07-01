package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/domain"
)

// ModelRepository defines the interface for model data access.
type ModelRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Model, error)
	GetByModelID(ctx context.Context, provider, modelID string) (*domain.Model, error)
	List(ctx context.Context, activeOnly bool) ([]*domain.Model, error)
	Create(ctx context.Context, model *domain.Model) error
	Update(ctx context.Context, model *domain.Model) error
}

type modelRepository struct {
	pool *pgxpool.Pool
}

// NewModelRepository creates a new model repository.
func NewModelRepository(pool *pgxpool.Pool) ModelRepository {
	return &modelRepository{pool: pool}
}

func (r *modelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Model, error) {
	query := `
		SELECT id, provider, model_id, display_name, description, context_window, max_output,
		       supports_streaming, supports_vision, supports_tools, input_price, output_price,
		       is_active, config, created_at, updated_at
		FROM models WHERE id = $1`

	m := &domain.Model{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Provider, &m.ModelID, &m.DisplayName, &m.Description,
		&m.ContextWindow, &m.MaxOutput, &m.SupportsStreaming, &m.SupportsVision,
		&m.SupportsTools, &m.InputPrice, &m.OutputPrice, &m.IsActive,
		&m.Config, &m.CreatedAt, &m.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("model not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return m, nil
}

func (r *modelRepository) GetByModelID(ctx context.Context, provider, modelID string) (*domain.Model, error) {
	query := `
		SELECT id, provider, model_id, display_name, description, context_window, max_output,
		       supports_streaming, supports_vision, supports_tools, input_price, output_price,
		       is_active, config, created_at, updated_at
		FROM models WHERE provider = $1 AND model_id = $2`

	m := &domain.Model{}
	err := r.pool.QueryRow(ctx, query, provider, modelID).Scan(
		&m.ID, &m.Provider, &m.ModelID, &m.DisplayName, &m.Description,
		&m.ContextWindow, &m.MaxOutput, &m.SupportsStreaming, &m.SupportsVision,
		&m.SupportsTools, &m.InputPrice, &m.OutputPrice, &m.IsActive,
		&m.Config, &m.CreatedAt, &m.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("model not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return m, nil
}

func (r *modelRepository) List(ctx context.Context, activeOnly bool) ([]*domain.Model, error) {
	query := `
		SELECT id, provider, model_id, display_name, description, context_window, max_output,
		       supports_streaming, supports_vision, supports_tools, input_price, output_price,
		       is_active, config, created_at, updated_at
		FROM models`

	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY provider, model_id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	models := make([]*domain.Model, 0)
	for rows.Next() {
		m := &domain.Model{}
		if err := rows.Scan(
			&m.ID, &m.Provider, &m.ModelID, &m.DisplayName, &m.Description,
			&m.ContextWindow, &m.MaxOutput, &m.SupportsStreaming, &m.SupportsVision,
			&m.SupportsTools, &m.InputPrice, &m.OutputPrice, &m.IsActive,
			&m.Config, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, m)
	}

	return models, nil
}

func (r *modelRepository) Create(ctx context.Context, model *domain.Model) error {
	query := `
		INSERT INTO models (id, provider, model_id, display_name, description, context_window, max_output,
		                    supports_streaming, supports_vision, supports_tools, input_price, output_price, is_active, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		model.ID, model.Provider, model.ModelID, model.DisplayName, model.Description,
		model.ContextWindow, model.MaxOutput, model.SupportsStreaming, model.SupportsVision,
		model.SupportsTools, model.InputPrice, model.OutputPrice, model.IsActive, model.Config,
	).Scan(&model.CreatedAt, &model.UpdatedAt)
}

func (r *modelRepository) Update(ctx context.Context, model *domain.Model) error {
	query := `
		UPDATE models SET display_name = $1, description = $2, context_window = $3, max_output = $4,
		                  supports_streaming = $5, supports_vision = $6, supports_tools = $7,
		                  input_price = $8, output_price = $9, is_active = $10, config = $11
		WHERE id = $12`

	tag, err := r.pool.Exec(ctx, query,
		model.DisplayName, model.Description, model.ContextWindow, model.MaxOutput,
		model.SupportsStreaming, model.SupportsVision, model.SupportsTools,
		model.InputPrice, model.OutputPrice, model.IsActive, model.Config, model.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("model not found")
	}
	return nil
}
