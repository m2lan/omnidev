package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/agent/internal/domain"
)

// StepRepository defines the interface for step data access.
type StepRepository interface {
	Create(ctx context.Context, step *domain.Step) error
	ListByRun(ctx context.Context, runID uuid.UUID) ([]domain.Step, error)
}

type stepRepository struct {
	pool *pgxpool.Pool
}

func NewStepRepository(pool *pgxpool.Pool) StepRepository {
	return &stepRepository{pool: pool}
}

func (r *stepRepository) Create(ctx context.Context, step *domain.Step) error {
	query := `
		INSERT INTO agent_steps (id, run_id, step_number, type, content, tool_name, tool_input, tool_output, status, error, token_input, token_output, latency_ms, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		step.ID, step.RunID, step.StepNumber, step.Type, step.Content,
		step.ToolName, step.ToolInput, step.ToolOutput, step.Status, step.Error,
		step.TokenInput, step.TokenOutput, step.LatencyMs, step.StartedAt, step.CompletedAt,
	).Scan(&step.CreatedAt)
}

func (r *stepRepository) ListByRun(ctx context.Context, runID uuid.UUID) ([]domain.Step, error) {
	query := `
		SELECT id, run_id, step_number, type, content, tool_name, tool_input, tool_output,
		       status, error, token_input, token_output, latency_ms, started_at, completed_at, created_at
		FROM agent_steps WHERE run_id = $1
		ORDER BY step_number ASC`

	rows, err := r.pool.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to list steps: %w", err)
	}
	defer rows.Close()

	steps := make([]domain.Step, 0)
	for rows.Next() {
		step := domain.Step{}
		if err := rows.Scan(
			&step.ID, &step.RunID, &step.StepNumber, &step.Type, &step.Content,
			&step.ToolName, &step.ToolInput, &step.ToolOutput, &step.Status, &step.Error,
			&step.TokenInput, &step.TokenOutput, &step.LatencyMs, &step.StartedAt, &step.CompletedAt,
			&step.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}
