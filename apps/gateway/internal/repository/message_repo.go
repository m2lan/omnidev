package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/gateway/internal/domain"
)

// MessageRepository defines the interface for message data access.
type MessageRepository interface {
	Create(ctx context.Context, msg *domain.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	ListByConversation(ctx context.Context, convID uuid.UUID, offset, limit int) ([]*domain.Message, int, error)
	GetRecentMessages(ctx context.Context, convID uuid.UUID, limit int) ([]*domain.Message, error)
}

type messageRepository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new message repository.
func NewMessageRepository(pool *pgxpool.Pool) MessageRepository {
	return &messageRepository{pool: pool}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (id, conversation_id, role, content, model_id, token_input, token_output, latency_ms, tool_calls, tool_call_id, parent_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.ModelID,
		msg.TokenInput, msg.TokenOutput, msg.LatencyMs, msg.ToolCalls,
		msg.ToolCallID, msg.ParentID, msg.Metadata,
	).Scan(&msg.CreatedAt)
}

func (r *messageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, model_id, token_input, token_output, latency_ms, tool_calls, tool_call_id, parent_id, metadata, created_at
		FROM messages
		WHERE id = $1`

	msg := &domain.Message{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.ModelID,
		&msg.TokenInput, &msg.TokenOutput, &msg.LatencyMs, &msg.ToolCalls,
		&msg.ToolCallID, &msg.ParentID, &msg.Metadata, &msg.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}
	return msg, nil
}

func (r *messageRepository) ListByConversation(ctx context.Context, convID uuid.UUID, offset, limit int) ([]*domain.Message, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, convID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	query := `
		SELECT id, conversation_id, role, content, model_id, token_input, token_output, latency_ms, tool_calls, tool_call_id, parent_id, metadata, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, convID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	msgs := make([]*domain.Message, 0)
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.ModelID,
			&msg.TokenInput, &msg.TokenOutput, &msg.LatencyMs, &msg.ToolCalls,
			&msg.ToolCallID, &msg.ParentID, &msg.Metadata, &msg.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		msgs = append(msgs, msg)
	}

	return msgs, total, nil
}

func (r *messageRepository) GetRecentMessages(ctx context.Context, convID uuid.UUID, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, model_id, token_input, token_output, latency_ms, tool_calls, tool_call_id, parent_id, metadata, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, convID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}
	defer rows.Close()

	msgs := make([]*domain.Message, 0)
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.ModelID,
			&msg.TokenInput, &msg.TokenOutput, &msg.LatencyMs, &msg.ToolCalls,
			&msg.ToolCallID, &msg.ParentID, &msg.Metadata, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		msgs = append(msgs, msg)
	}

	// Reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}
