// Package service contains the business logic for the Notification Service.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/notification/internal/channels"
	"github.com/omnidev/services/notification/internal/domain"
	"github.com/omnidev/services/notification/internal/repository"
)

// NotificationService handles notification operations.
type NotificationService struct {
	notifRepo  repository.NotificationRepository
	channelReg *channels.Registry
}

// NewNotificationService creates a new notification service.
func NewNotificationService(
	notifRepo repository.NotificationRepository,
	channelReg *channels.Registry,
) *NotificationService {
	return &NotificationService{
		notifRepo:  notifRepo,
		channelReg: channelReg,
	}
}

// SendNotificationInput defines the input for sending a notification.
type SendNotificationInput struct {
	UserID    string `json:"user_id" validate:"required"`
	Type      string `json:"type" validate:"required"`
	Title     string `json:"title" validate:"required"`
	Content   string `json:"content" validate:"required"`
	Channel   string `json:"channel"`
	ActionURL string `json:"action_url"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Send sends a notification.
func (s *NotificationService) Send(ctx context.Context, input *SendNotificationInput) (*domain.Notification, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, appErr.Validation("invalid user_id")
	}

	channel := domain.ChannelInApp
	if input.Channel != "" {
		channel = domain.NotificationChannel(input.Channel)
	}

	notif := &domain.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      domain.NotificationType(input.Type),
		Title:     input.Title,
		Content:   input.Content,
		Channel:   channel,
		Status:    domain.NotifStatusUnread,
		Metadata:  input.Metadata,
		CreatedAt: time.Now(),
	}

	if input.ActionURL != "" {
		notif.ActionURL = &input.ActionURL
	}
	if notif.Metadata == nil {
		notif.Metadata = map[string]interface{}{}
	}

	// Save to database
	if err := s.notifRepo.Create(ctx, notif); err != nil {
		return nil, appErr.Wrap(err, "failed to save notification")
	}

	// Deliver through channel
	ch, err := s.channelReg.Get(channel)
	if err != nil {
		logger.Log.Warn("Channel not found, notification saved but not delivered",
			zap.String("channel", string(channel)),
		)
		return notif, nil
	}

	if err := ch.Send(ctx, notif); err != nil {
		logger.Log.Error("Failed to deliver notification",
			zap.String("channel", string(channel)),
			zap.Error(err),
		)
		// Don't fail the request, notification is saved
	}

	return notif, nil
}

// ListNotifications returns notifications for a user.
func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.notifRepo.ListByUser(ctx, userID, offset, pageSize)
}

// MarkAsRead marks a notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, userID, notifID uuid.UUID) error {
	notif, err := s.notifRepo.GetByID(ctx, notifID)
	if err != nil {
		return appErr.NotFound("notification")
	}
	if notif.UserID != userID {
		return appErr.ErrForbidden
	}
	return s.notifRepo.MarkAsRead(ctx, notifID)
}

// MarkAllAsRead marks all notifications as read.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}

// UnreadCount returns the unread notification count.
func (s *NotificationService) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifRepo.UnreadCount(ctx, userID)
}

// GetPreferences returns notification preferences.
func (s *NotificationService) GetPreferences(ctx context.Context, userID uuid.UUID) ([]domain.NotificationPreference, error) {
	return s.notifRepo.GetPreferences(ctx, userID)
}

// UpdatePreferenceInput defines the input for updating a preference.
type UpdatePreferenceInput struct {
	Channel   string `json:"channel" validate:"required"`
	NotifType string `json:"notif_type" validate:"required"`
	Enabled   bool   `json:"enabled"`
}

// UpdatePreference updates a notification preference.
func (s *NotificationService) UpdatePreference(ctx context.Context, userID uuid.UUID, input *UpdatePreferenceInput) error {
	pref := &domain.NotificationPreference{
		ID:        uuid.New(),
		UserID:    userID,
		Channel:   domain.NotificationChannel(input.Channel),
		NotifType: domain.NotificationType(input.NotifType),
		Enabled:   input.Enabled,
	}
	return s.notifRepo.UpsertPreference(ctx, pref)
}
