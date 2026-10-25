// Package repository provides data access for the Chat Service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/chat/internal/domain"
)

// AttachmentRepository defines the interface for attachment data access.
type AttachmentRepository interface {
	Create(ctx context.Context, att *domain.Attachment) error
	GetByID(ctx context.Context, id string) (*domain.Attachment, error)
	ListByMessage(ctx context.Context, messageID string) ([]*domain.Attachment, error)
	ListByConversation(ctx context.Context, conversationID string) ([]*domain.Attachment, error)
	UpdateMessageID(ctx context.Context, ids []string, messageID string) error
	Delete(ctx context.Context, id string) error
}

// PostgresAttachmentRepository implements AttachmentRepository with PostgreSQL.
type PostgresAttachmentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAttachmentRepository creates a new PostgreSQL attachment repository.
func NewPostgresAttachmentRepository(pool *pgxpool.Pool) *PostgresAttachmentRepository {
	return &PostgresAttachmentRepository{pool: pool}
}

const attachmentColumns = `id, user_id, conversation_id, message_id, filename, mime_type,
	file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at`

func scanAttachment(row pgx.Row) (*domain.Attachment, error) {
	var att domain.Attachment
	err := row.Scan(
		&att.ID, &att.UserID, &att.ConversationID, &att.MessageID, &att.Filename, &att.MimeType,
		&att.FileSize, &att.StorageKey, &att.StorageURL, &att.ThumbnailKey, &att.Width, &att.Height,
		&att.Metadata, &att.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &att, nil
}

// Create inserts a new attachment record.
func (r *PostgresAttachmentRepository) Create(ctx context.Context, att *domain.Attachment) error {
	query := `
		INSERT INTO attachments (
			id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING created_at`

	err := r.pool.QueryRow(ctx, query,
		att.ID, att.UserID, att.ConversationID, att.MessageID, att.Filename, att.MimeType,
		att.FileSize, att.StorageKey, att.StorageURL, att.ThumbnailKey, att.Width, att.Height,
		att.Metadata,
	).Scan(&att.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	return nil
}

// GetByID returns an attachment by ID.
func (r *PostgresAttachmentRepository) GetByID(ctx context.Context, id string) (*domain.Attachment, error) {
	query := fmt.Sprintf(`SELECT %s FROM attachments WHERE id = $1 AND deleted_at IS NULL`, attachmentColumns)

	att, err := scanAttachment(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("attachment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return att, nil
}

// ListByMessage returns all attachments for a message.
func (r *PostgresAttachmentRepository) ListByMessage(ctx context.Context, messageID string) ([]*domain.Attachment, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM attachments
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`, attachmentColumns)

	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments by message: %w", err)
	}
	defer rows.Close()

	attachments := make([]*domain.Attachment, 0)
	for rows.Next() {
		att, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, att)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list attachments by message: %w", err)
	}
	return attachments, nil
}

// ListByConversation returns all attachments for a conversation.
func (r *PostgresAttachmentRepository) ListByConversation(ctx context.Context, conversationID string) ([]*domain.Attachment, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM attachments
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`, attachmentColumns)

	rows, err := r.pool.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments by conversation: %w", err)
	}
	defer rows.Close()

	attachments := make([]*domain.Attachment, 0)
	for rows.Next() {
		att, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, att)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list attachments by conversation: %w", err)
	}
	return attachments, nil
}

// UpdateMessageID updates the message_id for a batch of attachments.
func (r *PostgresAttachmentRepository) UpdateMessageID(ctx context.Context, ids []string, messageID string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE attachments
		SET message_id = $1
		WHERE id::text = ANY($2) AND deleted_at IS NULL`

	if _, err := r.pool.Exec(ctx, query, messageID, ids); err != nil {
		return fmt.Errorf("failed to update attachment message_id: %w", err)
	}
	return nil
}

// Delete soft-deletes an attachment.
func (r *PostgresAttachmentRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE attachments SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("attachment not found: %s", id)
	}
	return nil
}
