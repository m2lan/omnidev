// Package repository provides data access implementations for the Workflow Service.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/workflow/internal/domain"
)

// WorkflowRepository defines the interface for workflow data access.
type WorkflowRepository interface {
	Create(ctx context.Context, wf *domain.Workflow) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error)
	Update(ctx context.Context, id uuid.UUID, update *WorkflowUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Workflow, int, error)
}

type WorkflowUpdate struct {
	Name        *string
	Description *string
	Definition  *domain.WorkflowDefinition
	IsActive    *bool
	Tags        []string
}

type workflowRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowRepository(pool *pgxpool.Pool) WorkflowRepository {
	return &workflowRepository{pool: pool}
}

func (r *workflowRepository) Create(ctx context.Context, wf *domain.Workflow) error {
	query := `
		INSERT INTO workflows (id, user_id, org_id, name, description, definition, version, trigger_type, trigger_config, is_active, visibility, tags, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		wf.ID, wf.UserID, wf.OrgID, wf.Name, wf.Description,
		wf.Definition, wf.Version, wf.TriggerType, wf.TriggerConfig,
		wf.IsActive, wf.Visibility, wf.Tags, wf.Metadata,
	).Scan(&wf.CreatedAt, &wf.UpdatedAt)
}

func (r *workflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	query := `
		SELECT id, user_id, org_id, name, description, definition, version, trigger_type, trigger_config,
		       is_active, visibility, tags, metadata, created_at, updated_at
		FROM workflows WHERE id = $1 AND deleted_at IS NULL`

	wf := &domain.Workflow{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&wf.ID, &wf.UserID, &wf.OrgID, &wf.Name, &wf.Description,
		&wf.Definition, &wf.Version, &wf.TriggerType, &wf.TriggerConfig,
		&wf.IsActive, &wf.Visibility, &wf.Tags, &wf.Metadata,
		&wf.CreatedAt, &wf.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	return wf, nil
}

func (r *workflowRepository) Update(ctx context.Context, id uuid.UUID, update *WorkflowUpdate) error {
	setClauses := []string{"version = version + 1"}
	args := []interface{}{}
	argIdx := 1

	if update.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *update.Name)
		argIdx++
	}
	if update.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *update.Description)
		argIdx++
	}
	if update.Definition != nil {
		setClauses = append(setClauses, fmt.Sprintf("definition = $%d", argIdx))
		args = append(args, *update.Definition)
		argIdx++
	}
	if update.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *update.IsActive)
		argIdx++
	}
	if update.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, update.Tags)
		argIdx++
	}

	query := fmt.Sprintf("UPDATE workflows SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}

func (r *workflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE workflows SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}

func (r *workflowRepository) List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Workflow, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workflows WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workflows: %w", err)
	}

	query := `
		SELECT id, user_id, org_id, name, description, definition, version, trigger_type, trigger_config,
		       is_active, visibility, tags, metadata, created_at, updated_at
		FROM workflows WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	wfs := make([]*domain.Workflow, 0)
	for rows.Next() {
		wf := &domain.Workflow{}
		if err := rows.Scan(
			&wf.ID, &wf.UserID, &wf.OrgID, &wf.Name, &wf.Description,
			&wf.Definition, &wf.Version, &wf.TriggerType, &wf.TriggerConfig,
			&wf.IsActive, &wf.Visibility, &wf.Tags, &wf.Metadata,
			&wf.CreatedAt, &wf.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan workflow: %w", err)
		}
		wfs = append(wfs, wf)
	}

	return wfs, total, nil
}
