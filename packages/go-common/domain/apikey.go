package domain

import (
	"time"

	"github.com/google/uuid"
)

// APIKeyStatus represents the status of an API key.
type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         uuid.UUID    `json:"id" db:"id"`
	UserID     uuid.UUID    `json:"user_id" db:"user_id"`
	Name       string       `json:"name" db:"name"`
	KeyHash    string       `json:"-" db:"key_hash"`
	KeyPrefix  string       `json:"key_prefix" db:"key_prefix"`
	Scopes     []string     `json:"scopes" db:"scopes"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt *time.Time   `json:"last_used_at,omitempty" db:"last_used_at"`
	LastUsedIP *string      `json:"-" db:"last_used_ip"`
	Status     APIKeyStatus `json:"status" db:"status"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
}

// IsExpired returns true if the API key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid returns true if the API key is active and not expired.
func (k *APIKey) IsValid() bool {
	return k.Status == APIKeyStatusActive && !k.IsExpired()
}

// OAuthConnection represents an OAuth provider connection.
type OAuthConnection struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	Provider     string                 `json:"provider" db:"provider"`
	ProviderUID  string                 `json:"provider_uid" db:"provider_uid"`
	AccessToken  *string                `json:"-" db:"access_token"`
	RefreshToken *string                `json:"-" db:"refresh_token"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
	Scope        *string                `json:"scope,omitempty" db:"scope"`
	RawProfile   map[string]interface{} `json:"raw_profile,omitempty" db:"raw_profile"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}
