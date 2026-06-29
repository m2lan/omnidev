package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/workflow/internal/domain"
)

// NodeRunRepository defines the interface for node run data access.
type NodeRunRepository interface {
	Create(ctx context.Context, nodeRun *domain.NodeRun) error
	ListByRun(ctx context.Context, runID uuid.UUID) ([]domain.NodeRun, error)
}

type nodeRunRepository struct {
	pool *pgxpool.Pool
}

func NewNodeRunRepository(pool *pgxpool.Pool) NodeRunRepository {
	return &nodeRunRepository{pool: pool}
}

func (r *nodeRunRepository) Create(ctx context.Context, nodeRun *domain.NodeRun) error {
	query := `
		INSERT INTO workflow_node_runs (id, run_id, node_id, node_type, node_name, status, input, output, error, latency_ms, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		nodeRun.ID, nodeRun.RunID, nodeRun.NodeID, nodeRun.NodeType, nodeRun.NodeName,
		nodeRun.Status, nodeRun.Input, nodeRun.Output, nodeRun.Error,
		nodeRun.LatencyMs, nodeRun.StartedAt, nodeRun.CompletedAt,
	).Scan(&nodeRun.CreatedAt)
}

func (r *nodeRunRepository) ListByRun(ctx context.Context, runID uuid.UUID) ([]domain.NodeRun, error) {
	query := `
		SELECT id, run_id, node_id, node_type, node_name, status, input, output, error, retry_count, started_at, completed_at, latency_ms, created_at
		FROM workflow_node_runs WHERE run_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to list node runs: %w", err)
	}
	defer rows.Close()

	nodeRuns := make([]domain.NodeRun, 0)
	for rows.Next() {
		nr := domain.NodeRun{}
		if err := rows.Scan(
			&nr.ID, &nr.RunID, &nr.NodeID, &nr.NodeType, &nr.NodeName,
			&nr.Status, &nr.Input, &nr.Output, &nr.Error, &nr.RetryCount,
			&nr.StartedAt, &nr.CompletedAt, &nr.LatencyMs, &nr.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan node run: %w", err)
		}
		nodeRuns = append(nodeRuns, nr)
	}

	return nodeRuns, nil
}
