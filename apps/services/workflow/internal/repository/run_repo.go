package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/workflow/internal/domain"
)

// RunRepository defines the interface for workflow run data access.
type RunRepository interface {
	Create(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RunStatus, output map[string]interface{}, errMsg *string) error
	UpdateProgress(ctx context.Context, id uuid.UUID, completedNodes int) error
	ListByWorkflow(ctx context.Context, wfID uuid.UUID, offset, limit int) ([]*domain.Run, int, error)
}

type runRepository struct {
	pool *pgxpool.Pool
}

func NewRunRepository(pool *pgxpool.Pool) RunRepository {
	return &runRepository{pool: pool}
}

func (r *runRepository) Create(ctx context.Context, run *domain.Run) error {
	query := `
		INSERT INTO workflow_runs (id, workflow_id, user_id, workflow_version, status, trigger_type, input, total_nodes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		run.ID, run.WorkflowID, run.UserID, run.WorkflowVersion,
		run.Status, run.TriggerType, run.Input, run.TotalNodes, run.Metadata,
	).Scan(&run.CreatedAt)
}

func (r *runRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	query := `
		SELECT id, workflow_id, user_id, workflow_version, status, trigger_type, input, output, error,
		       total_nodes, completed_nodes, started_at, completed_at, metadata, created_at
		FROM workflow_runs WHERE id = $1`

	run := &domain.Run{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&run.ID, &run.WorkflowID, &run.UserID, &run.WorkflowVersion,
		&run.Status, &run.TriggerType, &run.Input, &run.Output, &run.Error,
		&run.TotalNodes, &run.CompletedNodes, &run.StartedAt, &run.CompletedAt,
		&run.Metadata, &run.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("run not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return run, nil
}

func (r *runRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RunStatus, output map[string]interface{}, errMsg *string) error {
	now := time.Now()
	var completedAt *time.Time
	var startedAt *time.Time
	if status == domain.RunStatusRunning {
		startedAt = &now
	}
	if status == domain.RunStatusSuccess || status == domain.RunStatusFailed || status == domain.RunStatusCancelled {
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = $1, output = $2, error = $3, started_at = COALESCE(started_at, $4), completed_at = $5 WHERE id = $6`,
		status, output, errMsg, startedAt, completedAt, id)
	return err
}

func (r *runRepository) UpdateProgress(ctx context.Context, id uuid.UUID, completedNodes int) error {
	_, err := r.pool.Exec(ctx, `UPDATE workflow_runs SET completed_nodes = $1 WHERE id = $2`, completedNodes, id)
	return err
}

func (r *runRepository) ListByWorkflow(ctx context.Context, wfID uuid.UUID, offset, limit int) ([]*domain.Run, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workflow_runs WHERE workflow_id = $1`, wfID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, workflow_id, user_id, workflow_version, status, trigger_type, input, output, error,
		       total_nodes, completed_nodes, started_at, completed_at, metadata, created_at
		FROM workflow_runs WHERE workflow_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, wfID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	runs := make([]*domain.Run, 0)
	for rows.Next() {
		run := &domain.Run{}
		if err := rows.Scan(
			&run.ID, &run.WorkflowID, &run.UserID, &run.WorkflowVersion,
			&run.Status, &run.TriggerType, &run.Input, &run.Output, &run.Error,
			&run.TotalNodes, &run.CompletedNodes, &run.StartedAt, &run.CompletedAt,
			&run.Metadata, &run.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}

	return runs, total, nil
}
