// Package repository provides data access implementations for the Gateway.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/go-common/domain"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, update *domain.UserUpdate) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *domain.UserFilter, offset, limit int) ([]*domain.User, int, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new user repository.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, email_verified, password_hash, nickname, avatar_url, bio, role, status, settings, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.EmailVerified, user.PasswordHash,
		user.Nickname, user.AvatarURL, user.Bio, user.Role, user.Status,
		user.Settings, user.Metadata,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("email already exists: %w", err)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, email_verified, password_hash, nickname, avatar_url, bio,
		       role, status, last_login_at, last_login_ip::text, settings, metadata,
		       created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash,
		&user.Nickname, &user.AvatarURL, &user.Bio, &user.Role, &user.Status,
		&user.LastLoginAt, &user.LastLoginIP, &user.Settings, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, email_verified, password_hash, nickname, avatar_url, bio,
		       role, status, last_login_at, last_login_ip::text, settings, metadata,
		       created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash,
		&user.Nickname, &user.AvatarURL, &user.Bio, &user.Role, &user.Status,
		&user.LastLoginAt, &user.LastLoginIP, &user.Settings, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", email)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, id uuid.UUID, update *domain.UserUpdate) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if update.Nickname != nil {
		setClauses = append(setClauses, fmt.Sprintf("nickname = $%d", argIdx))
		args = append(args, *update.Nickname)
		argIdx++
	}
	if update.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *update.AvatarURL)
		argIdx++
	}
	if update.Bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio = $%d", argIdx))
		args = append(args, *update.Bio)
		argIdx++
	}
	if update.Settings != nil {
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", argIdx))
		args = append(args, update.Settings)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `UPDATE users SET last_login_at = $1, last_login_ip = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, time.Now(), ip, id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	query := `UPDATE users SET status = $1 WHERE id = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = $1, status = 'deleted' WHERE id = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, filter *domain.UserFilter, offset, limit int) ([]*domain.User, int, error) {
	whereClauses := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.Status != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.Role != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", argIdx))
			args = append(args, *filter.Role)
			argIdx++
		}
		if filter.Search != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(email ILIKE $%d OR nickname ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+filter.Search+"%")
			argIdx++
		}
	}

	where := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Fetch page
	query := fmt.Sprintf(`
		SELECT id, email, email_verified, password_hash, nickname, avatar_url, bio,
		       role, status, last_login_at, last_login_ip::text, settings, metadata,
		       created_at, updated_at, deleted_at
		FROM users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		if err := rows.Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.PasswordHash,
			&user.Nickname, &user.AvatarURL, &user.Bio, &user.Role, &user.Status,
			&user.LastLoginAt, &user.LastLoginIP, &user.Settings, &user.Metadata,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (r *userRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return exists, nil
}
