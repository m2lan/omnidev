// Ownership: Will move to services/auth-service when microservices are extracted.
// Currently shared by gateway BFF layer.
package repository

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/go-common/domain"
)

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	GetByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID, ip net.IP) error
	Revoke(ctx context.Context, id uuid.UUID) error
}

type apiKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates a new API key repository.
func NewAPIKeyRepository(pool *pgxpool.Pool) APIKeyRepository {
	return &apiKeyRepository{pool: pool}
}

func (r *apiKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, scopes, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		key.ID, key.UserID, key.Name, key.KeyHash, key.KeyPrefix, key.Scopes, key.ExpiresAt, key.Status,
	).Scan(&key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, last_used_ip, status, created_at, updated_at
		FROM api_keys
		WHERE key_hash = $1 AND status = 'active'`

	key := &domain.APIKey{}
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.KeyPrefix,
		&key.Scopes, &key.ExpiresAt, &key.LastUsedAt, &key.LastUsedIP,
		&key.Status, &key.CreatedAt, &key.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return key, nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, last_used_ip, status, created_at, updated_at
		FROM api_keys
		WHERE id = $1`

	key := &domain.APIKey{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.KeyPrefix,
		&key.Scopes, &key.ExpiresAt, &key.LastUsedAt, &key.LastUsedIP,
		&key.Status, &key.CreatedAt, &key.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return key, nil
}

func (r *apiKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, last_used_ip, status, created_at, updated_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.APIKey, 0)
	for rows.Next() {
		key := &domain.APIKey{}
		if err := rows.Scan(
			&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.KeyPrefix,
			&key.Scopes, &key.ExpiresAt, &key.LastUsedAt, &key.LastUsedIP,
			&key.Status, &key.CreatedAt, &key.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID, ip net.IP) error {
	query := `UPDATE api_keys SET last_used_at = $1, last_used_ip = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, time.Now(), ip.String(), id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET status = 'revoked' WHERE id = $1 AND status = 'active'`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("API key not found or already revoked")
	}
	return nil
}
