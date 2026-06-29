// Package domain defines the core business entities for the Workflow Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// RunStatus represents the status of a workflow run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSuccess   RunStatus = "success"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// NodeRunStatus represents the status of a node run.
type NodeRunStatus string

const (
	NodeRunStatusPending NodeRunStatus = "pending"
	NodeRunStatusRunning NodeRunStatus = "running"
	NodeRunStatusSuccess NodeRunStatus = "success"
	NodeRunStatusFailed  NodeRunStatus = "failed"
	NodeRunStatusSkipped NodeRunStatus = "skipped"
)

// TriggerType represents how a workflow is triggered.
type TriggerType string

const (
	TriggerManual TriggerType = "manual"
	TriggerCron   TriggerType = "cron"
	TriggerWebhook TriggerType = "webhook"
)

// Workflow represents a workflow definition.
type Workflow struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	UserID        uuid.UUID              `json:"user_id" db:"user_id"`
	OrgID         *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Name          string                 `json:"name" db:"name"`
	Description   *string                `json:"description,omitempty" db:"description"`
	Definition    WorkflowDefinition     `json:"definition" db:"definition"`
	Version       int                    `json:"version" db:"version"`
	TriggerType   TriggerType            `json:"trigger_type" db:"trigger_type"`
	TriggerConfig map[string]interface{} `json:"trigger_config,omitempty" db:"trigger_config"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
	Visibility    string                 `json:"visibility" db:"visibility"`
	Tags          []string               `json:"tags" db:"tags"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time             `json:"-" db:"deleted_at"`
}

// WorkflowDefinition represents the structure of a workflow.
type WorkflowDefinition struct {
	Nodes []NodeDef `json:"nodes"`
	Edges []EdgeDef `json:"edges"`
}

// NodeDef represents a node in the workflow definition.
type NodeDef struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Position    Position               `json:"position"`
	Config      map[string]interface{} `json:"config"`
}

// EdgeDef represents an edge connecting two nodes.
type EdgeDef struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	SourceHandle string `json:"source_handle,omitempty"`
	Condition    string `json:"condition,omitempty"`
}

// Position represents x/y coordinates.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Run represents a workflow execution run.
type Run struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	WorkflowID     uuid.UUID              `json:"workflow_id" db:"workflow_id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	WorkflowVersion int                   `json:"workflow_version" db:"workflow_version"`
	Status         RunStatus              `json:"status" db:"status"`
	TriggerType    TriggerType            `json:"trigger_type" db:"trigger_type"`
	Input          map[string]interface{} `json:"input,omitempty" db:"input"`
	Output         map[string]interface{} `json:"output,omitempty" db:"output"`
	Error          *string                `json:"error,omitempty" db:"error"`
	TotalNodes     int                    `json:"total_nodes" db:"total_nodes"`
	CompletedNodes int                    `json:"completed_nodes" db:"completed_nodes"`
	StartedAt      *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// NodeRun represents the execution of a single node.
type NodeRun struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	RunID       uuid.UUID              `json:"run_id" db:"run_id"`
	NodeID      string                 `json:"node_id" db:"node_id"`
	NodeType    string                 `json:"node_type" db:"node_type"`
	NodeName    string                 `json:"node_name" db:"node_name"`
	Status      NodeRunStatus          `json:"status" db:"status"`
	Input       map[string]interface{} `json:"input,omitempty" db:"input"`
	Output      map[string]interface{} `json:"output,omitempty" db:"output"`
	Error       *string                `json:"error,omitempty" db:"error"`
	RetryCount  int                    `json:"retry_count" db:"retry_count"`
	StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	LatencyMs   *int                   `json:"latency_ms,omitempty" db:"latency_ms"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}
