// Package domain defines the core business entities for the MCP Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ServerStatus represents the status of an MCP server.
type ServerStatus string

const (
	ServerStatusActive  ServerStatus = "active"
	ServerStatusInactive ServerStatus = "inactive"
	ServerStatusError   ServerStatus = "error"
)

// TransportType represents the transport protocol.
type TransportType string

const (
	TransportSSE   TransportType = "sse"
	TransportStdio TransportType = "stdio"
)

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	OrgID       *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Name        string                 `json:"name" db:"name"`
	Description *string                `json:"description,omitempty" db:"description"`
	Transport   TransportType          `json:"transport" db:"transport"`
	Endpoint    *string                `json:"endpoint,omitempty" db:"endpoint"`
	Command     *string                `json:"command,omitempty" db:"command"`
	Args        []string               `json:"args,omitempty" db:"args"`
	Env         map[string]string      `json:"env,omitempty" db:"env"`
	IsBuiltin   bool                   `json:"is_builtin" db:"is_builtin"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	ToolCount   int                    `json:"tool_count" db:"tool_count"`
	LastHealthCheck *time.Time         `json:"last_health_check,omitempty" db:"last_health_check"`
	HealthStatus string                `json:"health_status" db:"health_status"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time             `json:"-" db:"deleted_at"`
}

// MCPTool represents a tool provided by an MCP server.
type MCPTool struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	ServerID    uuid.UUID              `json:"server_id" db:"server_id"`
	Name        string                 `json:"name" db:"name"`
	Description *string                `json:"description,omitempty" db:"description"`
	InputSchema map[string]interface{} `json:"input_schema" db:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty" db:"output_schema"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	CallCount   int64                  `json:"call_count" db:"call_count"`
	AvgLatencyMs *int                  `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ToolCallRequest represents a request to call an MCP tool.
type ToolCallRequest struct {
	ToolID uuid.UUID              `json:"tool_id" validate:"required"`
	Input  map[string]interface{} `json:"input"`
}

// ToolCallResponse represents the response from calling an MCP tool.
type ToolCallResponse struct {
	ToolID   uuid.UUID              `json:"tool_id"`
	Output   map[string]interface{} `json:"output"`
	Error    string                 `json:"error,omitempty"`
	Duration int                    `json:"duration_ms"`
}
