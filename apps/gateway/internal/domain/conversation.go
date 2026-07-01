// Package domain defines the core business entities for the Chat Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ConversationStatus represents the status of a conversation.
type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
)

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
)

// Model represents an AI model configuration.
type Model struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	Provider          string                 `json:"provider" db:"provider"`
	ModelID           string                 `json:"model_id" db:"model_id"`
	DisplayName       string                 `json:"display_name" db:"display_name"`
	Description       *string                `json:"description,omitempty" db:"description"`
	ContextWindow     int                    `json:"context_window" db:"context_window"`
	MaxOutput         int                    `json:"max_output" db:"max_output"`
	SupportsStreaming bool                   `json:"supports_streaming" db:"supports_streaming"`
	SupportsVision    bool                   `json:"supports_vision" db:"supports_vision"`
	SupportsTools     bool                   `json:"supports_tools" db:"supports_tools"`
	InputPrice        *float64               `json:"input_price,omitempty" db:"input_price"`
	OutputPrice       *float64               `json:"output_price,omitempty" db:"output_price"`
	IsActive          bool                   `json:"is_active" db:"is_active"`
	Config            map[string]interface{} `json:"config" db:"config"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// Conversation represents a chat conversation.
type Conversation struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	OrgID        *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Title        *string                `json:"title,omitempty" db:"title"`
	ModelID      *uuid.UUID             `json:"model_id,omitempty" db:"model_id"`
	SystemPrompt *string                `json:"system_prompt,omitempty" db:"system_prompt"`
	Settings     map[string]interface{} `json:"settings" db:"settings"`
	Status       ConversationStatus     `json:"status" db:"status"`
	Pinned       bool                   `json:"pinned" db:"pinned"`
	Tags         []string               `json:"tags" db:"tags"`
	MessageCount int                    `json:"message_count" db:"message_count"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time             `json:"-" db:"deleted_at"`
}

// Message represents a chat message.
type Message struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	ConversationID uuid.UUID              `json:"conversation_id" db:"conversation_id"`
	Role           MessageRole            `json:"role" db:"role"`
	Content        string                 `json:"content" db:"content"`
	ModelID        *string                `json:"model_id,omitempty" db:"model_id"`
	TokenInput     *int                   `json:"token_input,omitempty" db:"token_input"`
	TokenOutput    *int                   `json:"token_output,omitempty" db:"token_output"`
	LatencyMs      *int                   `json:"latency_ms,omitempty" db:"latency_ms"`
	ToolCalls      interface{}            `json:"tool_calls,omitempty" db:"tool_calls"`
	ToolCallID     *string                `json:"tool_call_id,omitempty" db:"tool_call_id"`
	ParentID       *uuid.UUID             `json:"parent_id,omitempty" db:"parent_id"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// PromptTemplate represents a reusable prompt template.
type PromptTemplate struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	OrgID       *uuid.UUID             `json:"org_id,omitempty" db:"org_id"`
	Title       string                 `json:"title" db:"title"`
	Content     string                 `json:"content" db:"content"`
	Description *string                `json:"description,omitempty" db:"description"`
	Category    *string                `json:"category,omitempty" db:"category"`
	Tags        []string               `json:"tags" db:"tags"`
	Variables   []interface{}          `json:"variables" db:"variables"`
	Visibility  string                 `json:"visibility" db:"visibility"`
	Version     int                    `json:"version" db:"version"`
	ForkFrom    *uuid.UUID             `json:"fork_from,omitempty" db:"fork_from"`
	UseCount    int                    `json:"use_count" db:"use_count"`
	LikeCount   int                    `json:"like_count" db:"like_count"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time             `json:"-" db:"deleted_at"`
}

// ChatChunk represents a streaming response chunk.
type ChatChunk struct {
	ID           string `json:"id"`
	Delta        string `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
	TokenInput   int    `json:"token_input,omitempty"`
	TokenOutput  int    `json:"token_output,omitempty"`
	ModelID      string `json:"model_id,omitempty"`
}
