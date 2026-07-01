package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/user/internal/domain"
)

// UserAIConfigRepository defines the interface for user AI config data access.
type UserAIConfigRepository interface {
	Create(ctx context.Context, config *domain.UserAIConfig) error
	Update(ctx context.Context, id uuid.UUID, update *domain.UserAIConfigUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.UserAIConfig, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, filter *domain.UserAIConfigFilter) ([]*domain.UserAIConfig, error)
	GetDefault(ctx context.Context, userID uuid.UUID) (*domain.UserAIConfig, error)
	ClearDefault(ctx context.Context, userID uuid.UUID) error
}

type userAIConfigRepository struct {
	pool *pgxpool.Pool
}

// NewUserAIConfigRepository creates a new user AI config repository.
func NewUserAIConfigRepository(pool *pgxpool.Pool) UserAIConfigRepository {
	return &userAIConfigRepository{pool: pool}
}

func (r *userAIConfigRepository) Create(ctx context.Context, config *domain.UserAIConfig) error {
	query := `
		INSERT INTO user_ai_configs (id, user_id, provider, display_name, api_key, base_url, protocol, models, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		config.ID, config.UserID, config.Provider, config.DisplayName,
		config.APIKey, config.BaseURL, config.Protocol, config.Models,
		config.IsDefault, config.IsActive,
	).Scan(&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user ai config: %w", err)
	}

	return nil
}

func (r *userAIConfigRepository) Update(ctx context.Context, id uuid.UUID, update *domain.UserAIConfigUpdate) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if update.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *update.DisplayName)
		argIdx++
	}
	if update.APIKey != nil {
		setClauses = append(setClauses, fmt.Sprintf("api_key = $%d", argIdx))
		args = append(args, *update.APIKey)
		argIdx++
	}
	if update.BaseURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("base_url = $%d", argIdx))
		args = append(args, *update.BaseURL)
		argIdx++
	}
	if update.Protocol != nil {
		setClauses = append(setClauses, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, string(*update.Protocol))
		argIdx++
	}
	if update.Models != nil {
		setClauses = append(setClauses, fmt.Sprintf("models = $%d", argIdx))
		args = append(args, update.Models)
		argIdx++
	}
	if update.IsDefault != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_default = $%d", argIdx))
		args = append(args, *update.IsDefault)
		argIdx++
	}
	if update.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *update.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE user_ai_configs SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user ai config: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user ai config not found: %s", id)
	}

	return nil
}

func (r *userAIConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_ai_configs SET deleted_at = $1, is_active = false WHERE id = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete user ai config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user ai config not found: %s", id)
	}
	return nil
}

func (r *userAIConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserAIConfig, error) {
	query := `
		SELECT id, user_id, provider, display_name, api_key, base_url, protocol, models,
		       is_default, is_active, created_at, updated_at, deleted_at
		FROM user_ai_configs
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanOne(r.pool.QueryRow(ctx, query, id))
}

func (r *userAIConfigRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.UserAIConfig, error) {
	query := `
		SELECT id, user_id, provider, display_name, api_key, base_url, protocol, models,
		       is_default, is_active, created_at, updated_at, deleted_at
		FROM user_ai_configs
		WHERE user_id = $1 AND provider = $2 AND deleted_at IS NULL AND is_active = true
		ORDER BY is_default DESC, created_at DESC
		LIMIT 1`

	return r.scanOne(r.pool.QueryRow(ctx, query, userID, provider))
}

func (r *userAIConfigRepository) ListByUserID(ctx context.Context, userID uuid.UUID, filter *domain.UserAIConfigFilter) ([]*domain.UserAIConfig, error) {
	whereClauses := []string{"user_id = $1", "deleted_at IS NULL"}
	args := []interface{}{userID}
	argIdx := 2

	if filter != nil {
		if filter.Provider != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("provider = $%d", argIdx))
			args = append(args, *filter.Provider)
			argIdx++
		}
		if filter.Active != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, *filter.Active)
			argIdx++
		}
	}

	where := strings.Join(whereClauses, " AND ")
	query := fmt.Sprintf(`
		SELECT id, user_id, provider, display_name, api_key, base_url, protocol, models,
		       is_default, is_active, created_at, updated_at, deleted_at
		FROM user_ai_configs
		WHERE %s
		ORDER BY is_default DESC, created_at DESC`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list user ai configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*domain.UserAIConfig, 0)
	for rows.Next() {
		config, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (r *userAIConfigRepository) GetDefault(ctx context.Context, userID uuid.UUID) (*domain.UserAIConfig, error) {
	query := `
		SELECT id, user_id, provider, display_name, api_key, base_url, protocol, models,
		       is_default, is_active, created_at, updated_at, deleted_at
		FROM user_ai_configs
		WHERE user_id = $1 AND is_default = true AND is_active = true AND deleted_at IS NULL
		LIMIT 1`

	return r.scanOne(r.pool.QueryRow(ctx, query, userID))
}

func (r *userAIConfigRepository) ClearDefault(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE user_ai_configs SET is_default = false WHERE user_id = $1 AND is_default = true AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear default: %w", err)
	}
	return nil
}

// scanOne scans a single row into a UserAIConfig.
func (r *userAIConfigRepository) scanOne(row pgx.Row) (*domain.UserAIConfig, error) {
	config := &domain.UserAIConfig{}
	var modelsJSON []byte

	err := row.Scan(
		&config.ID, &config.UserID, &config.Provider, &config.DisplayName,
		&config.APIKey, &config.BaseURL, &config.Protocol, &modelsJSON,
		&config.IsDefault, &config.IsActive,
		&config.CreatedAt, &config.UpdatedAt, &config.DeletedAt,
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

// scanRow scans a row set into a UserAIConfig.
func (r *userAIConfigRepository) scanRow(rows pgx.Rows) (*domain.UserAIConfig, error) {
	config := &domain.UserAIConfig{}
	var modelsJSON []byte

	err := rows.Scan(
		&config.ID, &config.UserID, &config.Provider, &config.DisplayName,
		&config.APIKey, &config.BaseURL, &config.Protocol, &modelsJSON,
		&config.IsDefault, &config.IsActive,
		&config.CreatedAt, &config.UpdatedAt, &config.DeletedAt,
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
