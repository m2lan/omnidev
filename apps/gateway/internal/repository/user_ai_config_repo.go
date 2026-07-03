package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/adapter"
)

const userAIConfigColumns = `
	id, user_id, provider, display_name, api_key, base_url, protocol, models,
	is_default, is_active, created_at, updated_at`

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
		SELECT ` + userAIConfigColumns + `
		FROM user_ai_configs
		WHERE user_id = $1 AND provider = $2 AND is_active = true AND deleted_at IS NULL
		ORDER BY is_default DESC, created_at DESC
		LIMIT 1`

	return r.scanOne(r.pool.QueryRow(ctx, query, userID, provider))
}

func (r *userAIConfigRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*adapter.UserAIConfig, error) {
	query := `
		SELECT ` + userAIConfigColumns + `
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
		SELECT ` + userAIConfigColumns + `
		FROM user_ai_configs
		WHERE id = $1 AND is_active = true AND deleted_at IS NULL`

	return r.scanOne(r.pool.QueryRow(ctx, query, id))
}

func (r *userAIConfigRepository) GetByIDAndUser(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*adapter.UserAIConfig, error) {
	query := `
		SELECT ` + userAIConfigColumns + `
		FROM user_ai_configs
		WHERE id = $1 AND user_id = $2 AND is_active = true AND deleted_at IS NULL`

	return r.scanOne(r.pool.QueryRow(ctx, query, id, userID))
}

func (r *userAIConfigRepository) Create(ctx context.Context, cfg *adapter.UserAIConfig, userID uuid.UUID) error {
	// Ensure models is not nil (database column is NOT NULL)
	models := cfg.Models
	if models == nil {
		models = []string{}
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	displayName := cfg.DisplayName
	if displayName == "" {
		displayName = cfg.Provider
	}

	query := `
		INSERT INTO user_ai_configs (user_id, provider, display_name, api_key, base_url, protocol, models, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id, created_at, updated_at`

	var createdAt, updatedAt time.Time
	err = r.pool.QueryRow(ctx, query,
		userID, cfg.Provider, displayName, cfg.APIKey, cfg.BaseURL, cfg.Protocol, modelsJSON,
	).Scan(&cfg.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user ai config: %w", err)
	}
	cfg.CreatedAt = createdAt.Format(time.RFC3339)
	cfg.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *userAIConfigRepository) Update(ctx context.Context, cfg *adapter.UserAIConfig) error {
	models := cfg.Models
	if models == nil {
		models = []string{}
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		UPDATE user_ai_configs
		SET provider = $2, display_name = $3, api_key = $4, base_url = $5, protocol = $6, models = $7, updated_at = NOW()
		WHERE id = $1 AND is_active = true AND deleted_at IS NULL
		RETURNING updated_at`

	var updatedAt time.Time
	err = r.pool.QueryRow(ctx, query, cfg.ID, cfg.Provider, cfg.DisplayName, cfg.APIKey, cfg.BaseURL, cfg.Protocol, modelsJSON).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user ai config: %w", err)
	}
	cfg.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
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
	_, err = tx.Exec(ctx, `UPDATE user_ai_configs SET is_default = false WHERE user_id = $1 AND deleted_at IS NULL`, userID)
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
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&config.ID, &config.UserID, &config.Provider, &config.DisplayName,
		&config.APIKey, &config.BaseURL, &config.Protocol, &modelsJSON,
		&config.IsDefault, &config.IsActive, &createdAt, &updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user ai config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user ai config: %w", err)
	}

	config.CreatedAt = createdAt.Format(time.RFC3339)
	config.UpdatedAt = updatedAt.Format(time.RFC3339)

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
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&config.ID, &config.UserID, &config.Provider, &config.DisplayName,
		&config.APIKey, &config.BaseURL, &config.Protocol, &modelsJSON,
		&config.IsDefault, &config.IsActive, &createdAt, &updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user ai config: %w", err)
	}

	config.CreatedAt = createdAt.Format(time.RFC3339)
	config.UpdatedAt = updatedAt.Format(time.RFC3339)

	if modelsJSON != nil {
		if err := json.Unmarshal(modelsJSON, &config.Models); err != nil {
			return nil, fmt.Errorf("failed to unmarshal models: %w", err)
		}
	}

	return config, nil
}
