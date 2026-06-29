package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/mcp/internal/domain"
)

// ToolRepository defines the interface for MCP tool data access.
type ToolRepository interface {
	Create(ctx context.Context, tool *domain.MCPTool) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPTool, error)
	ListByServer(ctx context.Context, serverID uuid.UUID) ([]domain.MCPTool, error)
	IncrementCallCount(ctx context.Context, id uuid.UUID, latencyMs int) error
}

type toolRepository struct {
	pool *pgxpool.Pool
}

func NewToolRepository(pool *pgxpool.Pool) ToolRepository {
	return &toolRepository{pool: pool}
}

func (r *toolRepository) Create(ctx context.Context, tool *domain.MCPTool) error {
	query := `
		INSERT INTO mcp_tools (id, server_id, name, description, input_schema, output_schema, is_active, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		tool.ID, tool.ServerID, tool.Name, tool.Description,
		tool.InputSchema, tool.OutputSchema, tool.IsActive, tool.Metadata,
	).Scan(&tool.CreatedAt, &tool.UpdatedAt)
}

func (r *toolRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPTool, error) {
	query := `
		SELECT id, server_id, name, description, input_schema, output_schema, is_active, call_count, avg_latency_ms, metadata, created_at, updated_at
		FROM mcp_tools WHERE id = $1`

	tool := &domain.MCPTool{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tool.ID, &tool.ServerID, &tool.Name, &tool.Description,
		&tool.InputSchema, &tool.OutputSchema, &tool.IsActive,
		&tool.CallCount, &tool.AvgLatencyMs, &tool.Metadata,
		&tool.CreatedAt, &tool.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	return tool, nil
}

func (r *toolRepository) ListByServer(ctx context.Context, serverID uuid.UUID) ([]domain.MCPTool, error) {
	query := `
		SELECT id, server_id, name, description, input_schema, output_schema, is_active, call_count, avg_latency_ms, metadata, created_at, updated_at
		FROM mcp_tools WHERE server_id = $1 AND is_active = true
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	defer rows.Close()

	tools := make([]domain.MCPTool, 0)
	for rows.Next() {
		tool := domain.MCPTool{}
		if err := rows.Scan(
			&tool.ID, &tool.ServerID, &tool.Name, &tool.Description,
			&tool.InputSchema, &tool.OutputSchema, &tool.IsActive,
			&tool.CallCount, &tool.AvgLatencyMs, &tool.Metadata,
			&tool.CreatedAt, &tool.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool: %w", err)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

func (r *toolRepository) IncrementCallCount(ctx context.Context, id uuid.UUID, latencyMs int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE mcp_tools SET call_count = call_count + 1, avg_latency_ms = (COALESCE(avg_latency_ms, 0) * call_count + $1) / (call_count + 1) WHERE id = $2`,
		latencyMs, id)
	return err
}
