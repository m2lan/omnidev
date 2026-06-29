// Package channels provides notification delivery channel implementations.
package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/services/notification/internal/domain"
)

// Channel defines the interface for notification delivery channels.
type Channel interface {
	// Name returns the channel name.
	Name() string

	// Send sends a notification through this channel.
	Send(ctx context.Context, notif *domain.Notification) error
}

// Registry manages notification channels.
type Registry struct {
	channels map[domain.NotificationChannel]Channel
}

// NewRegistry creates a new channel registry.
func NewRegistry() *Registry {
	return &Registry{
		channels: make(map[domain.NotificationChannel]Channel),
	}
}

// Register adds a channel.
func (r *Registry) Register(ch Channel) {
	r.channels[domain.NotificationChannel(ch.Name())] = ch
}

// Get returns a channel by name.
func (r *Registry) Get(name domain.NotificationChannel) (Channel, error) {
	ch, ok := r.channels[name]
	if !ok {
		return nil, fmt.Errorf("channel not found: %s", name)
	}
	return ch, nil
}

// InAppChannel handles in-app notifications.
type InAppChannel struct{}

func NewInAppChannel() *InAppChannel { return &InAppChannel{} }
func (c *InAppChannel) Name() string { return "in_app" }

func (c *InAppChannel) Send(ctx context.Context, notif *domain.Notification) error {
	// In-app notifications are stored in the database
	// No external delivery needed
	logger.Log.Debug("In-app notification created",
		zap.String("user_id", notif.UserID.String()),
		zap.String("title", notif.Title),
	)
	return nil
}

// EmailChannel sends notifications via email.
type EmailChannel struct {
	smtpHost string
	smtpPort int
	from     string
}

func NewEmailChannel() *EmailChannel {
	return &EmailChannel{
		smtpHost: "localhost",
		smtpPort: 587,
		from:     "noreply@omnidev.dev",
	}
}

func (c *EmailChannel) Name() string { return "email" }

func (c *EmailChannel) Send(ctx context.Context, notif *domain.Notification) error {
	// TODO: Integrate with SMTP
	logger.Log.Info("Email notification sent",
		zap.String("user_id", notif.UserID.String()),
		zap.String("title", notif.Title),
	)
	return nil
}

// SlackChannel sends notifications to Slack.
type SlackChannel struct {
	webhookURL string
	client     *http.Client
}

func NewSlackChannel() *SlackChannel {
	return &SlackChannel{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *SlackChannel) Name() string { return "slack" }

func (c *SlackChannel) Send(ctx context.Context, notif *domain.Notification) error {
	if c.webhookURL == "" {
		logger.Log.Debug("Slack webhook not configured, skipping")
		return nil
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf("*%s*\n%s", notif.Title, notif.Content),
	}

	data, _ := json.Marshal(payload)
	resp, err := c.client.Post(c.webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API error: %d", resp.StatusCode)
	}

	logger.Log.Info("Slack notification sent",
		zap.String("user_id", notif.UserID.String()),
		zap.String("title", notif.Title),
	)
	return nil
}

// WebhookChannel sends notifications to a webhook URL.
type WebhookChannel struct {
	client *http.Client
}

func NewWebhookChannel() *WebhookChannel {
	return &WebhookChannel{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *WebhookChannel) Name() string { return "webhook" }

func (c *WebhookChannel) Send(ctx context.Context, notif *domain.Notification) error {
	webhookURL, ok := notif.Metadata["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url not set in notification metadata")
	}

	payload := map[string]interface{}{
		"id":         notif.ID.String(),
		"type":       notif.Type,
		"title":      notif.Title,
		"content":    notif.Content,
		"created_at": notif.CreatedAt,
	}

	data, _ := json.Marshal(payload)
	resp, err := c.client.Post(webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	logger.Log.Info("Webhook notification sent",
		zap.String("user_id", notif.UserID.String()),
		zap.String("webhook_url", webhookURL),
	)
	return nil
}
