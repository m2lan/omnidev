package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/agent/internal/domain"
)

// RunRepository defines the interface for run data access.
type RunRepository interface {
	Create(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RunStatus, result, errMsg *string) error
	UpdateProgress(ctx context.Context, id uuid.UUID, completedSteps int) error
	ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*domain.Run, int, error)
}

type runRepository struct {
	pool *pgxpool.Pool
}

func NewRunRepository(pool *pgxpool.Pool) RunRepository {
	return &runRepository{pool: pool}
}

func (r *runRepository) Create(ctx context.Context, run *domain.Run) error {
	query := `
		INSERT INTO agent_runs (id, agent_id, user_id, task, status, total_steps, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		run.ID, run.AgentID, run.UserID, run.Task, run.Status, run.TotalSteps, run.Metadata,
	).Scan(&run.CreatedAt, &run.UpdatedAt)
}

func (r *runRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	query := `
		SELECT id, agent_id, user_id, task, status, result, error,
		       total_steps, completed_steps, token_input, token_output, cost,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM agent_runs WHERE id = $1`

	run := &domain.Run{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&run.ID, &run.AgentID, &run.UserID, &run.Task, &run.Status, &run.Result, &run.Error,
		&run.TotalSteps, &run.CompletedSteps, &run.TokenInput, &run.TokenOutput, &run.Cost,
		&run.StartedAt, &run.CompletedAt, &run.Metadata, &run.CreatedAt, &run.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("run not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return run, nil
}

func (r *runRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RunStatus, result, errMsg *string) error {
	now := time.Now()
	var completedAt *time.Time
	if status == domain.RunStatusSuccess || status == domain.RunStatusFailed || status == domain.RunStatusCancelled {
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE agent_runs SET status = $1, result = $2, error = $3, completed_at = $4 WHERE id = $5`,
		status, result, errMsg, completedAt, id)
	return err
}

func (r *runRepository) UpdateProgress(ctx context.Context, id uuid.UUID, completedSteps int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE agent_runs SET completed_steps = $1 WHERE id = $2`,
		completedSteps, id)
	return err
}

func (r *runRepository) ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*domain.Run, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE agent_id = $1`, agentID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count runs: %w", err)
	}

	query := `
		SELECT id, agent_id, user_id, task, status, result, error,
		       total_steps, completed_steps, token_input, token_output, cost,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM agent_runs WHERE agent_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, agentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	runs := make([]*domain.Run, 0)
	for rows.Next() {
		run := &domain.Run{}
		if err := rows.Scan(
			&run.ID, &run.AgentID, &run.UserID, &run.Task, &run.Status, &run.Result, &run.Error,
			&run.TotalSteps, &run.CompletedSteps, &run.TokenInput, &run.TokenOutput, &run.Cost,
			&run.StartedAt, &run.CompletedAt, &run.Metadata, &run.CreatedAt, &run.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan run: %w", err)
		}
		runs = append(runs, run)
	}

	return runs, total, nil
}
