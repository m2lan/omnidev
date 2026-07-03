// Package repository provides data access for the Chat Service.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
	db *sqlx.DB
}

// NewPostgresAttachmentRepository creates a new PostgreSQL attachment repository.
func NewPostgresAttachmentRepository(db *sqlx.DB) *PostgresAttachmentRepository {
	return &PostgresAttachmentRepository{db: db}
}

// Create inserts a new attachment record.
func (r *PostgresAttachmentRepository) Create(ctx context.Context, att *domain.Attachment) error {
	query := `
		INSERT INTO attachments (
			id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata
		) VALUES (
			:id, :user_id, :conversation_id, :message_id, :filename, :mime_type,
			:file_size, :storage_key, :storage_url, :thumbnail_key, :width, :height, :metadata
		)`

	_, err := r.db.NamedExecContext(ctx, query, att)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			return fmt.Errorf("database error (code %s): %w", pqErr.Code, pqErr)
		}
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	return nil
}

// GetByID returns an attachment by ID.
func (r *PostgresAttachmentRepository) GetByID(ctx context.Context, id string) (*domain.Attachment, error) {
	var att domain.Attachment
	query := `
		SELECT id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at
		FROM attachments
		WHERE id = $1 AND deleted_at IS NULL`

	if err := r.db.GetContext(ctx, &att, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("attachment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return &att, nil
}

// ListByMessage returns all attachments for a message.
func (r *PostgresAttachmentRepository) ListByMessage(ctx context.Context, messageID string) ([]*domain.Attachment, error) {
	var attachments []*domain.Attachment
	query := `
		SELECT id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at
		FROM attachments
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`

	if err := r.db.SelectContext(ctx, &attachments, query, messageID); err != nil {
		return nil, fmt.Errorf("failed to list attachments by message: %w", err)
	}
	return attachments, nil
}

// ListByConversation returns all attachments for a conversation.
func (r *PostgresAttachmentRepository) ListByConversation(ctx context.Context, conversationID string) ([]*domain.Attachment, error) {
	var attachments []*domain.Attachment
	query := `
		SELECT id, user_id, conversation_id, message_id, filename, mime_type,
			file_size, storage_key, storage_url, thumbnail_key, width, height, metadata, created_at
		FROM attachments
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &attachments, query, conversationID); err != nil {
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
		SET message_id = $1, updated_at = NOW()
		WHERE id = ANY($2) AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, messageID, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to update attachment message_id: %w", err)
	}
	return nil
}

// Delete soft-deletes an attachment.
func (r *PostgresAttachmentRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE attachments SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("attachment not found: %s", id)
	}
	return nil
}
