// Package repository provides data access implementations for the MCP Service.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/mcp/internal/domain"
)

// ServerRepository defines the interface for MCP server data access.
type ServerRepository interface {
	Create(ctx context.Context, server *domain.MCPServer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPServer, error)
	Update(ctx context.Context, id uuid.UUID, update *ServerUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.MCPServer, int, error)
	UpdateHealth(ctx context.Context, id uuid.UUID, status string) error
}

type ServerUpdate struct {
	Name        *string
	Description *string
	Endpoint    *string
	IsActive    *bool
}

type serverRepository struct {
	pool *pgxpool.Pool
}

func NewServerRepository(pool *pgxpool.Pool) ServerRepository {
	return &serverRepository{pool: pool}
}

func (r *serverRepository) Create(ctx context.Context, server *domain.MCPServer) error {
	query := `
		INSERT INTO mcp_servers (id, user_id, org_id, name, description, transport, endpoint, command, args, env, is_builtin, is_active, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		server.ID, server.UserID, server.OrgID, server.Name, server.Description,
		server.Transport, server.Endpoint, server.Command, server.Args, server.Env,
		server.IsBuiltin, server.IsActive, server.Metadata,
	).Scan(&server.CreatedAt, &server.UpdatedAt)
}

func (r *serverRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPServer, error) {
	query := `
		SELECT id, user_id, org_id, name, description, transport, endpoint, command, args, env,
		       is_builtin, is_active, tool_count, last_health_check, health_status, metadata, created_at, updated_at
		FROM mcp_servers WHERE id = $1 AND deleted_at IS NULL`

	server := &domain.MCPServer{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&server.ID, &server.UserID, &server.OrgID, &server.Name, &server.Description,
		&server.Transport, &server.Endpoint, &server.Command, &server.Args, &server.Env,
		&server.IsBuiltin, &server.IsActive, &server.ToolCount, &server.LastHealthCheck,
		&server.HealthStatus, &server.Metadata, &server.CreatedAt, &server.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("server not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	return server, nil
}

func (r *serverRepository) Update(ctx context.Context, id uuid.UUID, update *ServerUpdate) error {
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
	if update.Endpoint != nil {
		setClauses = append(setClauses, fmt.Sprintf("endpoint = $%d", argIdx))
		args = append(args, *update.Endpoint)
		argIdx++
	}
	if update.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *update.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE mcp_servers SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}
	return nil
}

func (r *serverRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE mcp_servers SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}
	return nil
}

func (r *serverRepository) List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.MCPServer, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM mcp_servers WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	query := `
		SELECT id, user_id, org_id, name, description, transport, endpoint, command, args, env,
		       is_builtin, is_active, tool_count, last_health_check, health_status, metadata, created_at, updated_at
		FROM mcp_servers WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY is_builtin DESC, created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	servers := make([]*domain.MCPServer, 0)
	for rows.Next() {
		server := &domain.MCPServer{}
		if err := rows.Scan(
			&server.ID, &server.UserID, &server.OrgID, &server.Name, &server.Description,
			&server.Transport, &server.Endpoint, &server.Command, &server.Args, &server.Env,
			&server.IsBuiltin, &server.IsActive, &server.ToolCount, &server.LastHealthCheck,
			&server.HealthStatus, &server.Metadata, &server.CreatedAt, &server.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, total, nil
}

func (r *serverRepository) UpdateHealth(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE mcp_servers SET health_status = $1, last_health_check = $2 WHERE id = $3`,
		status, time.Now(), id)
	return err
}
