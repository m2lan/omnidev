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

// UserAIConfigRepository defines the interface for fetching user AI configs.
type UserAIConfigRepository interface {
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*adapter.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*adapter.UserAIConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*adapter.UserAIConfig, error)
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
