// Package domain defines the core business entities for the User Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AIProtocol represents the API protocol used by an AI provider.
type AIProtocol string

const (
	AIProtocolOpenAI    AIProtocol = "openai"
	AIProtocolAnthropic AIProtocol = "anthropic"
)

// UserAIConfig represents a user's custom AI provider configuration.
type UserAIConfig struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	UserID      uuid.UUID    `json:"user_id" db:"user_id"`
	Provider    string       `json:"provider" db:"provider"`
	DisplayName string       `json:"display_name" db:"display_name"`
	APIKey      string       `json:"-" db:"api_key"` // Sensitive, never expose in JSON
	BaseURL     string       `json:"base_url" db:"base_url"`
	Protocol    AIProtocol   `json:"protocol" db:"protocol"`
	Models      []ModelConfig `json:"models" db:"models"`
	IsDefault   bool         `json:"is_default" db:"is_default"`
	IsActive    bool         `json:"is_active" db:"is_active"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time   `json:"-" db:"deleted_at"`
}

// ModelConfig represents a model within a user's AI configuration.
type ModelConfig struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"display_name"`
	DefaultTemperature *float64 `json:"default_temperature,omitempty"`
	DefaultMaxTokens   *int     `json:"default_max_tokens,omitempty"`
	ContextWindow      *int     `json:"context_window,omitempty"`
}

// IsDeleted returns true if the config has been soft-deleted.
func (c *UserAIConfig) IsDeleted() bool { return c.DeletedAt != nil }

// UserAIConfigFilter defines filters for querying user AI configs.
type UserAIConfigFilter struct {
	Provider *string
	Active   *bool
}

// UserAIConfigUpdate defines fields that can be updated.
type UserAIConfigUpdate struct {
	DisplayName *string        `json:"display_name,omitempty"`
	APIKey      *string        `json:"api_key,omitempty"`
	BaseURL     *string        `json:"base_url,omitempty"`
	Protocol    *AIProtocol    `json:"protocol,omitempty"`
	Models      []ModelConfig  `json:"models,omitempty"`
	IsDefault   *bool          `json:"is_default,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
}
