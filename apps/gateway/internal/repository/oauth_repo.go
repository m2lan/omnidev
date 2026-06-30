package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/go-common/domain"
)

// OAuthRepository defines the interface for OAuth connection data access.
type OAuthRepository interface {
	Create(ctx context.Context, conn *domain.OAuthConnection) error
	GetByProvider(ctx context.Context, provider, providerUID string) (*domain.OAuthConnection, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, provider string) (*domain.OAuthConnection, error)
	Update(ctx context.Context, conn *domain.OAuthConnection) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type oauthRepository struct {
	pool *pgxpool.Pool
}

// NewOAuthRepository creates a new OAuth repository.
func NewOAuthRepository(pool *pgxpool.Pool) OAuthRepository {
	return &oauthRepository{pool: pool}
}

func (r *oauthRepository) Create(ctx context.Context, conn *domain.OAuthConnection) error {
	query := `
		INSERT INTO oauth_connections (id, user_id, provider, provider_uid, access_token, refresh_token, expires_at, scope, raw_profile)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		conn.ID, conn.UserID, conn.Provider, conn.ProviderUID,
		conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope, conn.RawProfile,
	).Scan(&conn.CreatedAt, &conn.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create OAuth connection: %w", err)
	}
	return nil
}

func (r *oauthRepository) GetByProvider(ctx context.Context, provider, providerUID string) (*domain.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_uid, access_token, refresh_token, expires_at, scope, raw_profile, created_at, updated_at
		FROM oauth_connections
		WHERE provider = $1 AND provider_uid = $2`

	conn := &domain.OAuthConnection{}
	err := r.pool.QueryRow(ctx, query, provider, providerUID).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderUID,
		&conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.Scope, &conn.RawProfile,
		&conn.CreatedAt, &conn.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("OAuth connection not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth connection: %w", err)
	}
	return conn, nil
}

func (r *oauthRepository) GetByUserID(ctx context.Context, userID uuid.UUID, provider string) (*domain.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_uid, access_token, refresh_token, expires_at, scope, raw_profile, created_at, updated_at
		FROM oauth_connections
		WHERE user_id = $1 AND provider = $2`

	conn := &domain.OAuthConnection{}
	err := r.pool.QueryRow(ctx, query, userID, provider).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderUID,
		&conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.Scope, &conn.RawProfile,
		&conn.CreatedAt, &conn.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("OAuth connection not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth connection: %w", err)
	}
	return conn, nil
}

func (r *oauthRepository) Update(ctx context.Context, conn *domain.OAuthConnection) error {
	query := `
		UPDATE oauth_connections
		SET access_token = $1, refresh_token = $2, expires_at = $3, scope = $4, raw_profile = $5
		WHERE id = $6`

	_, err := r.pool.Exec(ctx, query,
		conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope, conn.RawProfile, conn.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update OAuth connection: %w", err)
	}
	return nil
}

func (r *oauthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth_connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth connection: %w", err)
	}
	return nil
}
