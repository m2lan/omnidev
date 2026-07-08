// Package repository provides data access implementations for the Gateway.
//
// Ownership: Will move to services/conversation-service when microservices are extracted.
// Currently shared by gateway BFF layer.
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/domain"
)

// ConversationRepository defines the interface for conversation data access.
type ConversationRepository interface {
	Create(ctx context.Context, conv *domain.Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, update *ConversationUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, filter *ConversationFilter, offset, limit int) ([]*domain.Conversation, int, error)
	IncrementMessageCount(ctx context.Context, id uuid.UUID) error
}

// ConversationUpdate defines fields that can be updated.
type ConversationUpdate struct {
	Title            *string
	ModelID          *uuid.UUID
	SystemPrompt     *string
	Status           *domain.ConversationStatus
	Pinned           *bool
	Tags             []string
	KnowledgeBaseIDs *[]uuid.UUID
}

// ConversationFilter defines filters for listing conversations.
type ConversationFilter struct {
	Status  *domain.ConversationStatus
	ModelID *uuid.UUID
	Search  string
}

type conversationRepository struct {
	pool *pgxpool.Pool
}

// NewConversationRepository creates a new conversation repository.
func NewConversationRepository(pool *pgxpool.Pool) ConversationRepository {
	return &conversationRepository{pool: pool}
}

func (r *conversationRepository) Create(ctx context.Context, conv *domain.Conversation) error {
	query := `
		INSERT INTO conversations (id, user_id, org_id, title, model_id, system_prompt, settings, status, pinned, tags, knowledge_base_ids, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at, message_count`

	return r.pool.QueryRow(ctx, query,
		conv.ID, conv.UserID, conv.OrgID, conv.Title, conv.ModelID,
		conv.SystemPrompt, conv.Settings, conv.Status, conv.Pinned, conv.Tags, conv.KnowledgeBaseIDs, conv.Metadata,
	).Scan(&conv.CreatedAt, &conv.UpdatedAt, &conv.MessageCount)
}

func (r *conversationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	query := `
		SELECT id, user_id, org_id, title, model_id, system_prompt, settings, status, pinned, tags, knowledge_base_ids, message_count, metadata, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND deleted_at IS NULL`

	conv := &domain.Conversation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.UserID, &conv.OrgID, &conv.Title, &conv.ModelID,
		&conv.SystemPrompt, &conv.Settings, &conv.Status, &conv.Pinned,
		&conv.Tags, &conv.KnowledgeBaseIDs, &conv.MessageCount, &conv.Metadata, &conv.CreatedAt, &conv.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	return conv, nil
}

func (r *conversationRepository) Update(ctx context.Context, id uuid.UUID, update *ConversationUpdate) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if update.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *update.Title)
		argIdx++
	}
	if update.ModelID != nil {
		setClauses = append(setClauses, fmt.Sprintf("model_id = $%d", argIdx))
		args = append(args, *update.ModelID)
		argIdx++
	}
	if update.SystemPrompt != nil {
		setClauses = append(setClauses, fmt.Sprintf("system_prompt = $%d", argIdx))
		args = append(args, *update.SystemPrompt)
		argIdx++
	}
	if update.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *update.Status)
		argIdx++
	}
	if update.Pinned != nil {
		setClauses = append(setClauses, fmt.Sprintf("pinned = $%d", argIdx))
		args = append(args, *update.Pinned)
		argIdx++
	}
	if update.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, update.Tags)
		argIdx++
	}
	if update.KnowledgeBaseIDs != nil {
		setClauses = append(setClauses, fmt.Sprintf("knowledge_base_ids = $%d", argIdx))
		args = append(args, *update.KnowledgeBaseIDs)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE conversations SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE conversations SET deleted_at = NOW(), status = 'archived' WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *conversationRepository) List(ctx context.Context, userID uuid.UUID, filter *ConversationFilter, offset, limit int) ([]*domain.Conversation, int, error) {
	whereClauses := []string{"user_id = $1", "deleted_at IS NULL"}
	args := []interface{}{userID}
	argIdx := 2

	if filter != nil {
		if filter.Status != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.ModelID != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("model_id = $%d", argIdx))
			args = append(args, *filter.ModelID)
			argIdx++
		}
		if filter.Search != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("title ILIKE $%d", argIdx))
			args = append(args, "%"+filter.Search+"%")
			argIdx++
		}
	}

	where := strings.Join(whereClauses, " AND ")

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM conversations WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	// Fetch
	query := fmt.Sprintf(`
		SELECT id, user_id, org_id, title, model_id, system_prompt, settings, status, pinned, tags, knowledge_base_ids, message_count, metadata, created_at, updated_at
		FROM conversations
		WHERE %s
		ORDER BY pinned DESC, updated_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	convs := make([]*domain.Conversation, 0)
	for rows.Next() {
		conv := &domain.Conversation{}
		if err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.OrgID, &conv.Title, &conv.ModelID,
			&conv.SystemPrompt, &conv.Settings, &conv.Status, &conv.Pinned,
			&conv.Tags, &conv.KnowledgeBaseIDs, &conv.MessageCount, &conv.Metadata, &conv.CreatedAt, &conv.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convs = append(convs, conv)
	}

	return convs, total, nil
}

func (r *conversationRepository) IncrementMessageCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE conversations SET message_count = message_count + 1 WHERE id = $1`, id)
	return err
}
