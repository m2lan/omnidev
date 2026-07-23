// Package domain defines the core business entities for the Agent Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// RunStatus represents the status of an agent run.
type RunStatus string

const (
	RunStatusCreated    RunStatus = "created"
	RunStatusPlanning   RunStatus = "planning"
	RunStatusExecuting  RunStatus = "executing"
	RunStatusWaitingTool RunStatus = "waiting_tool"
	RunStatusSuccess    RunStatus = "success"
	RunStatusFailed     RunStatus = "failed"
	RunStatusCancelled  RunStatus = "cancelled"
)

// StepType represents the type of an agent step.
type StepType string

const (
	StepTypeThink     StepType = "think"
	StepTypeToolCall  StepType = "tool_call"
	StepTypeCodeExec  StepType = "code_exec"
	StepTypeResponse  StepType = "response"
	StepTypePlan      StepType = "plan"
)

// StepStatus represents the status of a step.
type StepStatus string

const (
	StepStatusPending StepStatus = "pending"
	StepStatusRunning StepStatus = "running"
	StepStatusSuccess StepStatus = "success"
	StepStatusFailed  StepStatus = "failed"
	StepStatusSkipped StepStatus = "skipped"
)

// Agent represents an AI agent configuration.
type Agent struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	OrgID        *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Name         string                 `json:"name" db:"name"`
	Description  *string                `json:"description,omitempty" db:"description"`
	AvatarURL    *string                `json:"avatar_url,omitempty" db:"avatar_url"`
	SystemPrompt string                 `json:"system_prompt" db:"system_prompt"`
	ModelID      *uuid.UUID             `json:"model_id,omitempty" db:"model_id"`
	Tools        []ToolConfig           `json:"tools" db:"tools"`
	MCPServers   []MCPServerConfig      `json:"mcp_servers" db:"mcp_servers"`
	Config       AgentConfig            `json:"config" db:"config"`
	Visibility   string                 `json:"visibility" db:"visibility"`
	IsTemplate   bool                   `json:"is_template" db:"is_template"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time             `json:"-" db:"deleted_at"`
}

// ToolConfig represents a tool configuration.
type ToolConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// MCPServerConfig represents an MCP server configuration.
type MCPServerConfig struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Transport string `json:"transport"` // sse, stdio
}

// AgentConfig holds agent-specific settings.
type AgentConfig struct {
	MaxSteps       int      `json:"max_steps"`
	MaxTokens      int      `json:"max_tokens"`
	Temperature    float64  `json:"temperature"`
	ToolChoice     string   `json:"tool_choice"` // auto, none, required
	ResponseFormat string   `json:"response_format"`
}

// Run represents an agent execution run.
type Run struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	AgentID        uuid.UUID              `json:"agent_id" db:"agent_id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	Task           string                 `json:"task" db:"task"`
	Status         RunStatus              `json:"status" db:"status"`
	Result         *string                `json:"result,omitempty" db:"result"`
	Error          *string                `json:"error,omitempty" db:"error"`
	TotalSteps     int                    `json:"total_steps" db:"total_steps"`
	CompletedSteps int                    `json:"completed_steps" db:"completed_steps"`
	TokenInput     int                    `json:"token_input" db:"token_input"`
	TokenOutput    int                    `json:"token_output" db:"token_output"`
	Cost           float64                `json:"cost" db:"cost"`
	StartedAt      *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// Step represents a single step in an agent run.
type Step struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	RunID       uuid.UUID              `json:"run_id" db:"run_id"`
	StepNumber  int                    `json:"step_number" db:"step_number"`
	Type        StepType               `json:"type" db:"type"`
	Content     *string                `json:"content,omitempty" db:"content"`
	ToolName    *string                `json:"tool_name,omitempty" db:"tool_name"`
	ToolInput   map[string]interface{} `json:"tool_input,omitempty" db:"tool_input"`
	ToolOutput  map[string]interface{} `json:"tool_output,omitempty" db:"tool_output"`
	Status      StepStatus             `json:"status" db:"status"`
	Error       *string                `json:"error,omitempty" db:"error"`
	TokenInput  *int                   `json:"token_input,omitempty" db:"token_input"`
	TokenOutput *int                   `json:"token_output,omitempty" db:"token_output"`
	LatencyMs   *int                   `json:"latency_ms,omitempty" db:"latency_ms"`
	StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Input    map[string]interface{} `json:"input"`
	Output   map[string]interface{} `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int                    `json:"duration_ms,omitempty"`
}
