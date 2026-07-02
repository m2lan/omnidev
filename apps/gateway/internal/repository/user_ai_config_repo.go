package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/adapter"
)

// UserAIConfigRepository defines the interface for user AI config operations.
type UserAIConfigRepository interface {
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*adapter.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*adapter.UserAIConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*adapter.UserAIConfig, error)
	GetByIDAndUser(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*adapter.UserAIConfig, error)
	Create(ctx context.Context, cfg *adapter.UserAIConfig, userID uuid.UUID) error
	Update(ctx context.Context, cfg *adapter.UserAIConfig) error
	Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	SetDefault(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}

type userAIConfigRepository struct {
	pool *pgxpool.Pool
}

// NewUserAIConfigRepository creates a new user AI config repository.
func NewUserAIConfigRepository(pool *pgxpool.Pool) UserAIConfigRepository {
	return &userAIConfigRepository{pool: pool}
}

func (r *userAIConfigRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*adapter.UserAIConfig, error) {
	query := `
		SELECT id, provider, api_key, base_url, protocol, models
		FROM user_ai_configs
		WHERE user_id = $1 AND provider = $2 AND is_active = true AND deleted_at IS NULL
		ORDER BY is_default DESC, created_at DESC
		LIMIT 1`

	return r.scanOne(r.pool.QueryRow(ctx, query, userID, provider))
}

func (r *userAIConfigRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*adapter.UserAIConfig, error) {
	query := `
		SELECT id, provider, api_key, base_url, protocol, models
		FROM user_ai_configs
		WHERE user_id = $1 AND is_active = true AND deleted_at IS NULL
		ORDER BY is_default DESC, created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user ai configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*adapter.UserAIConfig, 0)
	for rows.Next() {
		config, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (r *userAIConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*adapter.UserAIConfig, error) {
	query := `
		SELECT id, provider, api_key, base_url, protocol, models
		FROM user_ai_configs
		WHERE id = $1 AND is_active = true AND deleted_at IS NULL`

	return r.scanOne(r.pool.QueryRow(ctx, query, id))
}

func (r *userAIConfigRepository) GetByIDAndUser(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*adapter.UserAIConfig, error) {
	query := `
		SELECT id, provider, api_key, base_url, protocol, models
		FROM user_ai_configs
		WHERE id = $1 AND user_id = $2 AND is_active = true AND deleted_at IS NULL`

	return r.scanOne(r.pool.QueryRow(ctx, query, id, userID))
}

func (r *userAIConfigRepository) Create(ctx context.Context, cfg *adapter.UserAIConfig, userID uuid.UUID) error {
	modelsJSON, err := json.Marshal(cfg.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		INSERT INTO user_ai_configs (user_id, provider, display_name, api_key, base_url, protocol, models, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id`

	return r.pool.QueryRow(ctx, query,
		userID, cfg.Provider, cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Protocol, modelsJSON,
	).Scan(&cfg.ID)
}

func (r *userAIConfigRepository) Update(ctx context.Context, cfg *adapter.UserAIConfig) error {
	modelsJSON, err := json.Marshal(cfg.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		UPDATE user_ai_configs
		SET provider = $2, api_key = $3, base_url = $4, protocol = $5, models = $6, updated_at = NOW()
		WHERE id = $1 AND is_active = true AND deleted_at IS NULL`

	_, err = r.pool.Exec(ctx, query, cfg.ID, cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Protocol, modelsJSON)
	return err
}

func (r *userAIConfigRepository) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	query := `
		UPDATE user_ai_configs
		SET deleted_at = NOW(), is_active = false
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user ai config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user ai config not found")
	}
	return nil
}

func (r *userAIConfigRepository) SetDefault(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing default
	_, err = tx.Exec(ctx, `UPDATE user_ai_configs SET is_default = false WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to clear default: %w", err)
	}

	// Set new default
	tag, err := tx.Exec(ctx, `UPDATE user_ai_configs SET is_default = true WHERE id = $1 AND user_id = $2 AND is_active = true AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user ai config not found")
	}

	return tx.Commit(ctx)
}

func (r *userAIConfigRepository) scanOne(row pgx.Row) (*adapter.UserAIConfig, error) {
	config := &adapter.UserAIConfig{}
	var modelsJSON []byte

	err := row.Scan(
		&config.ID, &config.Provider, &config.APIKey,
		&config.BaseURL, &config.Protocol, &modelsJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user ai config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user ai config: %w", err)
	}

	if modelsJSON != nil {
		if err := json.Unmarshal(modelsJSON, &config.Models); err != nil {
			return nil, fmt.Errorf("failed to unmarshal models: %w", err)
		}
	}

	return config, nil
}

func (r *userAIConfigRepository) scanRow(rows pgx.Rows) (*adapter.UserAIConfig, error) {
	config := &adapter.UserAIConfig{}
	var modelsJSON []byte

	err := rows.Scan(
		&config.ID, &config.Provider, &config.APIKey,
		&config.BaseURL, &config.Protocol, &modelsJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user ai config: %w", err)
	}

	if modelsJSON != nil {
		if err := json.Unmarshal(modelsJSON, &config.Models); err != nil {
			return nil, fmt.Errorf("failed to unmarshal models: %w", err)
		}
	}

	return config, nil
}
