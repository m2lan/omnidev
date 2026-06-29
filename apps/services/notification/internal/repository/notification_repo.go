// Package repository provides data access implementations for the Notification Service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/notification/internal/domain"
)

// NotificationRepository defines the interface for notification data access.
type NotificationRepository interface {
	Create(ctx context.Context, notif *domain.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)

	// Preferences
	GetPreferences(ctx context.Context, userID uuid.UUID) ([]domain.NotificationPreference, error)
	UpsertPreference(ctx context.Context, pref *domain.NotificationPreference) error
}

type notificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{pool: pool}
}

func (r *notificationRepository) Create(ctx context.Context, notif *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, title, content, channel, status, action_url, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		notif.ID, notif.UserID, notif.Type, notif.Title,
		notif.Content, notif.Channel, notif.Status,
		notif.ActionURL, notif.Metadata,
	).Scan(&notif.CreatedAt)
}

func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	query := `
		SELECT id, user_id, type, title, content, channel, status, read_at, action_url, metadata, created_at
		FROM notifications WHERE id = $1`

	notif := &domain.Notification{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&notif.ID, &notif.UserID, &notif.Type, &notif.Title,
		&notif.Content, &notif.Channel, &notif.Status,
		&notif.ReadAt, &notif.ActionURL, &notif.Metadata, &notif.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("notification not found: %w", err)
	}
	return notif, nil
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, type, title, content, channel, status, read_at, action_url, metadata, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notifs := make([]*domain.Notification, 0)
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title,
			&n.Content, &n.Channel, &n.Status,
			&n.ReadAt, &n.ActionURL, &n.Metadata, &n.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		notifs = append(notifs, n)
	}

	return notifs, total, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status = 'read', read_at = $1 WHERE id = $2`,
		now, id)
	return err
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status = 'read', read_at = $1 WHERE user_id = $2 AND status = 'unread'`,
		now, userID)
	return err
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND status = 'unread'`,
		userID).Scan(&count)
	return count, err
}

func (r *notificationRepository) GetPreferences(ctx context.Context, userID uuid.UUID) ([]domain.NotificationPreference, error) {
	query := `
		SELECT id, user_id, channel, notif_type, enabled, created_at, updated_at
		FROM notification_preferences WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefs := make([]domain.NotificationPreference, 0)
	for rows.Next() {
		p := domain.NotificationPreference{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Channel, &p.NotifType, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}

	return prefs, nil
}

func (r *notificationRepository) UpsertPreference(ctx context.Context, pref *domain.NotificationPreference) error {
	query := `
		INSERT INTO notification_preferences (id, user_id, channel, notif_type, enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, channel, notif_type) DO UPDATE SET enabled = $5, updated_at = NOW()
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		pref.ID, pref.UserID, pref.Channel, pref.NotifType, pref.Enabled,
	).Scan(&pref.CreatedAt, &pref.UpdatedAt)
}
