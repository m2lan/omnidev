package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/chat/internal/domain"
)

// PromptRepository defines the interface for prompt template data access.
type PromptRepository interface {
	Create(ctx context.Context, prompt *domain.PromptTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PromptTemplate, error)
	Update(ctx context.Context, id uuid.UUID, update *PromptUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, filter *PromptFilter, offset, limit int) ([]*domain.PromptTemplate, int, error)
	IncrementUseCount(ctx context.Context, id uuid.UUID) error
}

// PromptUpdate defines fields that can be updated.
type PromptUpdate struct {
	Title       *string
	Content     *string
	Description *string
	Category    *string
	Tags        []string
	Visibility  *string
}

// PromptFilter defines filters for listing prompts.
type PromptFilter struct {
	Visibility *string
	Category   *string
	Search     string
}

type promptRepository struct {
	pool *pgxpool.Pool
}

// NewPromptRepository creates a new prompt repository.
func NewPromptRepository(pool *pgxpool.Pool) PromptRepository {
	return &promptRepository{pool: pool}
}

func (r *promptRepository) Create(ctx context.Context, prompt *domain.PromptTemplate) error {
	query := `
		INSERT INTO prompt_templates (id, user_id, org_id, title, content, description, category, tags, variables, visibility, fork_from, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING version, use_count, like_count, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		prompt.ID, prompt.UserID, prompt.OrgID, prompt.Title, prompt.Content,
		prompt.Description, prompt.Category, prompt.Tags, prompt.Variables,
		prompt.Visibility, prompt.ForkFrom, prompt.Metadata,
	).Scan(&prompt.Version, &prompt.UseCount, &prompt.LikeCount, &prompt.CreatedAt, &prompt.UpdatedAt)
}

func (r *promptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PromptTemplate, error) {
	query := `
		SELECT id, user_id, org_id, title, content, description, category, tags, variables,
		       visibility, version, fork_from, use_count, like_count, metadata, created_at, updated_at
		FROM prompt_templates
		WHERE id = $1 AND deleted_at IS NULL`

	p := &domain.PromptTemplate{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.OrgID, &p.Title, &p.Content, &p.Description,
		&p.Category, &p.Tags, &p.Variables, &p.Visibility, &p.Version,
		&p.ForkFrom, &p.UseCount, &p.LikeCount, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("prompt not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}
	return p, nil
}

func (r *promptRepository) Update(ctx context.Context, id uuid.UUID, update *PromptUpdate) error {
	setClauses := []string{"version = version + 1"}
	args := []interface{}{}
	argIdx := 1

	if update.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *update.Title)
		argIdx++
	}
	if update.Content != nil {
		setClauses = append(setClauses, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *update.Content)
		argIdx++
	}
	if update.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *update.Description)
		argIdx++
	}
	if update.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *update.Category)
		argIdx++
	}
	if update.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, update.Tags)
		argIdx++
	}
	if update.Visibility != nil {
		setClauses = append(setClauses, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, *update.Visibility)
		argIdx++
	}

	query := fmt.Sprintf("UPDATE prompt_templates SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("prompt not found")
	}
	return nil
}

func (r *promptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE prompt_templates SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("prompt not found")
	}
	return nil
}

func (r *promptRepository) List(ctx context.Context, userID uuid.UUID, filter *PromptFilter, offset, limit int) ([]*domain.PromptTemplate, int, error) {
	whereClauses := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	// Show user's own + public prompts
	whereClauses = append(whereClauses, fmt.Sprintf("(user_id = $%d OR visibility = 'public')", argIdx))
	args = append(args, userID)
	argIdx++

	if filter != nil {
		if filter.Visibility != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("visibility = $%d", argIdx))
			args = append(args, *filter.Visibility)
			argIdx++
		}
		if filter.Category != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIdx))
			args = append(args, *filter.Category)
			argIdx++
		}
		if filter.Search != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+filter.Search+"%")
			argIdx++
		}
	}

	where := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM prompt_templates WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count prompts: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, org_id, title, content, description, category, tags, variables,
		       visibility, version, fork_from, use_count, like_count, metadata, created_at, updated_at
		FROM prompt_templates
		WHERE %s
		ORDER BY use_count DESC, created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list prompts: %w", err)
	}
	defer rows.Close()

	prompts := make([]*domain.PromptTemplate, 0)
	for rows.Next() {
		p := &domain.PromptTemplate{}
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.OrgID, &p.Title, &p.Content, &p.Description,
			&p.Category, &p.Tags, &p.Variables, &p.Visibility, &p.Version,
			&p.ForkFrom, &p.UseCount, &p.LikeCount, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan prompt: %w", err)
		}
		prompts = append(prompts, p)
	}

	return prompts, total, nil
}

func (r *promptRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE prompt_templates SET use_count = use_count + 1 WHERE id = $1`, id)
	return err
}
