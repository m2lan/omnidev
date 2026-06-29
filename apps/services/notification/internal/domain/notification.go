// Package domain defines the core business entities for the Notification Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification.
type NotificationType string

const (
	NotifTypeSystem   NotificationType = "system"
	NotifTypeBilling  NotificationType = "billing"
	NotifTypeAgent    NotificationType = "agent"
	NotifTypeDeploy   NotificationType = "deploy"
	NotifTypeWorkflow NotificationType = "workflow"
	NotifTypeChat     NotificationType = "chat"
)

// NotificationChannel represents the delivery channel.
type NotificationChannel string

const (
	ChannelInApp    NotificationChannel = "in_app"
	ChannelEmail    NotificationChannel = "email"
	ChannelSlack    NotificationChannel = "slack"
	ChannelWebhook  NotificationChannel = "webhook"
)

// NotificationStatus represents the status of a notification.
type NotificationStatus string

const (
	NotifStatusUnread   NotificationStatus = "unread"
	NotifStatusRead     NotificationStatus = "read"
	NotifStatusArchived NotificationStatus = "archived"
)

// Notification represents a notification.
type Notification struct {
	ID        uuid.UUID           `json:"id" db:"id"`
	UserID    uuid.UUID           `json:"user_id" db:"user_id"`
	Type      NotificationType    `json:"type" db:"type"`
	Title     string              `json:"title" db:"title"`
	Content   string              `json:"content" db:"content"`
	Channel   NotificationChannel `json:"channel" db:"channel"`
	Status    NotificationStatus  `json:"status" db:"status"`
	ReadAt    *time.Time          `json:"read_at,omitempty" db:"read_at"`
	ActionURL *string             `json:"action_url,omitempty" db:"action_url"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
}

// NotificationPreference represents user notification preferences.
type NotificationPreference struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Channel   NotificationChannel `json:"channel" db:"channel"`
	NotifType NotificationType    `json:"notif_type" db:"notif_type"`
	Enabled   bool                `json:"enabled" db:"enabled"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt time.Time           `json:"updated_at" db:"updated_at"`
}
