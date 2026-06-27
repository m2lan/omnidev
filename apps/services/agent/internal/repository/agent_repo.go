// Package repository provides data access implementations for the Agent Service.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/agent/internal/domain"
)

// AgentRepository defines the interface for agent data access.
type AgentRepository interface {
	Create(ctx context.Context, agent *domain.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	Update(ctx context.Context, id uuid.UUID, update *AgentUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Agent, int, error)
}

type AgentUpdate struct {
	Name         *string
	Description  *string
	SystemPrompt *string
	Tools        []domain.ToolConfig
	Config       *domain.AgentConfig
}

type agentRepository struct {
	pool *pgxpool.Pool
}

func NewAgentRepository(pool *pgxpool.Pool) AgentRepository {
	return &agentRepository{pool: pool}
}

func (r *agentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	query := `
		INSERT INTO agents (id, user_id, org_id, name, description, avatar_url, system_prompt, model_id, tools, mcp_servers, config, visibility, is_template, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		agent.ID, agent.UserID, agent.OrgID, agent.Name, agent.Description,
		agent.AvatarURL, agent.SystemPrompt, agent.ModelID,
		agent.Tools, agent.MCPServers, agent.Config,
		agent.Visibility, agent.IsTemplate, agent.Metadata,
	).Scan(&agent.CreatedAt, &agent.UpdatedAt)
}

func (r *agentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	query := `
		SELECT id, user_id, org_id, name, description, avatar_url, system_prompt, model_id,
		       tools, mcp_servers, config, visibility, is_template, metadata, created_at, updated_at
		FROM agents WHERE id = $1 AND deleted_at IS NULL`

	agent := &domain.Agent{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&agent.ID, &agent.UserID, &agent.OrgID, &agent.Name, &agent.Description,
		&agent.AvatarURL, &agent.SystemPrompt, &agent.ModelID,
		&agent.Tools, &agent.MCPServers, &agent.Config,
		&agent.Visibility, &agent.IsTemplate, &agent.Metadata,
		&agent.CreatedAt, &agent.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("agent not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return agent, nil
}

func (r *agentRepository) Update(ctx context.Context, id uuid.UUID, update *AgentUpdate) error {
	setClauses := []string{}
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
	if update.SystemPrompt != nil {
		setClauses = append(setClauses, fmt.Sprintf("system_prompt = $%d", argIdx))
		args = append(args, *update.SystemPrompt)
		argIdx++
	}
	if update.Tools != nil {
		setClauses = append(setClauses, fmt.Sprintf("tools = $%d", argIdx))
		args = append(args, update.Tools)
		argIdx++
	}
	if update.Config != nil {
		setClauses = append(setClauses, fmt.Sprintf("config = $%d", argIdx))
		args = append(args, *update.Config)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE agents SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

func (r *agentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE agents SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

func (r *agentRepository) List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Agent, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	query := `
		SELECT id, user_id, org_id, name, description, avatar_url, system_prompt, model_id,
		       tools, mcp_servers, config, visibility, is_template, metadata, created_at, updated_at
		FROM agents WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	agents := make([]*domain.Agent, 0)
	for rows.Next() {
		agent := &domain.Agent{}
		if err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.OrgID, &agent.Name, &agent.Description,
			&agent.AvatarURL, &agent.SystemPrompt, &agent.ModelID,
			&agent.Tools, &agent.MCPServers, &agent.Config,
			&agent.Visibility, &agent.IsTemplate, &agent.Metadata,
			&agent.CreatedAt, &agent.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, total, nil
}
