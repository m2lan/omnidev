// Package repository provides data access for the Gateway.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/domain"
)

// AttachmentRepository defines the interface for attachment data access.
type AttachmentRepository interface {
	Create(ctx context.Context, att *domain.Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	ListByMessage(ctx context.Context, messageID uuid.UUID) ([]*domain.Attachment, error)
	UpdateMessageID(ctx context.Context, ids []uuid.UUID, messageID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type attachmentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAttachmentRepository creates a new PostgreSQL attachment repository.
func NewPostgresAttachmentRepository(pool *pgxpool.Pool) AttachmentRepository {
	return &attachmentRepository{pool: pool}
}

func (r *attachmentRepository) Create(ctx context.Context, att *domain.Attachment) error {
	query := `
		INSERT INTO attachments (
			id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		att.ID, att.UserID, att.ConversationID, att.MessageID,
		att.Filename, att.MimeType, att.FileSize,
		att.StorageKey, att.StorageURL, att.ThumbnailKey,
		att.Width, att.Height, att.Metadata,
	).Scan(&att.CreatedAt)
}

func (r *attachmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	query := `
		SELECT id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at
		FROM attachments
		WHERE id = $1 AND deleted_at IS NULL`

	att := &domain.Attachment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&att.ID, &att.UserID, &att.ConversationID, &att.MessageID,
		&att.Filename, &att.MimeType, &att.FileSize,
		&att.StorageKey, &att.StorageURL, &att.ThumbnailKey,
		&att.Width, &att.Height, &att.Metadata, &att.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("attachment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return att, nil
}

func (r *attachmentRepository) ListByMessage(ctx context.Context, messageID uuid.UUID) ([]*domain.Attachment, error) {
	query := `
		SELECT id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at
		FROM attachments
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*domain.Attachment
	for rows.Next() {
		att := &domain.Attachment{}
		if err := rows.Scan(
			&att.ID, &att.UserID, &att.ConversationID, &att.MessageID,
			&att.Filename, &att.MimeType, &att.FileSize,
			&att.StorageKey, &att.StorageURL, &att.ThumbnailKey,
			&att.Width, &att.Height, &att.Metadata, &att.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, att)
	}
	return attachments, nil
}

func (r *attachmentRepository) UpdateMessageID(ctx context.Context, ids []uuid.UUID, messageID uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query := `UPDATE attachments SET message_id = $1 WHERE id = ANY($2) AND deleted_at IS NULL`

	_, err := r.pool.Exec(ctx, query, messageID, ids)
	if err != nil {
		return fmt.Errorf("failed to update attachment message_id: %w", err)
	}
	return nil
}

func (r *attachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE attachments SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}
