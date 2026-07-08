package service

import (
	"context"

	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"

	"github.com/omnidev/gateway/internal/domain"
	"github.com/omnidev/gateway/internal/repository"
)

// ConversationService handles conversation CRUD and message listing.
type ConversationService struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
	attRepo  repository.AttachmentRepository
}

// NewConversationService creates a new conversation service.
func NewConversationService(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	attRepo repository.AttachmentRepository,
) *ConversationService {
	return &ConversationService{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		attRepo:  attRepo,
	}
}

// CreateConversationInput defines the input for creating a conversation.
type CreateConversationInput struct {
	Title            string                 `json:"title"`
	ModelID          string                 `json:"model_id"`
	SystemPrompt     string                 `json:"system_prompt"`
	Tags             []string               `json:"tags"`
	Settings         map[string]interface{} `json:"settings"`
	KnowledgeBaseIDs []string               `json:"knowledge_base_ids"`
}

// parseUUIDs converts string slice to uuid.UUID slice, skipping invalid entries.
func parseUUIDs(strs []string) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		if u, err := uuid.Parse(s); err == nil {
			ids = append(ids, u)
		}
	}
	return ids
}

// CreateConversation creates a new conversation.
func (s *ConversationService) CreateConversation(ctx context.Context, userID uuid.UUID, input *CreateConversationInput) (*domain.Conversation, error) {
	conv := &domain.Conversation{
		ID:               uuid.New(),
		UserID:           userID,
		Status:           domain.ConversationStatusActive,
		Tags:             input.Tags,
		Settings:         input.Settings,
		KnowledgeBaseIDs: parseUUIDs(input.KnowledgeBaseIDs),
		Metadata:         map[string]interface{}{},
	}

	if input.Title != "" {
		conv.Title = &input.Title
	}
	if input.SystemPrompt != "" {
		conv.SystemPrompt = &input.SystemPrompt
	}
	if input.Settings == nil {
		conv.Settings = map[string]interface{}{}
	}
	if input.Tags == nil {
		conv.Tags = []string{}
	}
	if input.KnowledgeBaseIDs == nil {
		conv.KnowledgeBaseIDs = []uuid.UUID{}
	}

	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, appErr.Wrap(err, "failed to create conversation")
	}

	return conv, nil
}

// GetConversation returns a conversation by ID with ownership verification.
func (s *ConversationService) GetConversation(ctx context.Context, userID, convID uuid.UUID) (*domain.Conversation, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, appErr.NotFound("conversation")
	}

	if conv.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	return conv, nil
}

// ListConversationsInput defines filters for listing conversations.
type ListConversationsInput struct {
	Status   string `form:"status"`
	ModelID  string `form:"model_id"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListConversations returns a paginated list of conversations.
func (s *ConversationService) ListConversations(ctx context.Context, userID uuid.UUID, input *ListConversationsInput) ([]*domain.Conversation, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	filter := &repository.ConversationFilter{
		Search: input.Search,
	}

	if input.Status != "" {
		status := domain.ConversationStatus(input.Status)
		filter.Status = &status
	}

	offset := (input.Page - 1) * input.PageSize
	return s.convRepo.List(ctx, userID, filter, offset, input.PageSize)
}

// UpdateConversationInput defines fields for updating a conversation.
type UpdateConversationInput struct {
	Title            *string  `json:"title"`
	SystemPrompt     *string  `json:"system_prompt"`
	Status           *string  `json:"status"`
	Pinned           *bool    `json:"pinned"`
	KnowledgeBaseIDs *[]string `json:"knowledge_base_ids"`
}

// UpdateConversation updates a conversation with ownership verification.
func (s *ConversationService) UpdateConversation(ctx context.Context, userID, convID uuid.UUID, input *UpdateConversationInput) (*domain.Conversation, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	update := &repository.ConversationUpdate{
		Title:            input.Title,
		SystemPrompt:     input.SystemPrompt,
		Pinned:           input.Pinned,
	}
	if input.KnowledgeBaseIDs != nil {
		parsed := parseUUIDs(*input.KnowledgeBaseIDs)
		update.KnowledgeBaseIDs = &parsed
	}
	if input.Status != nil {
		status := domain.ConversationStatus(*input.Status)
		update.Status = &status
	}

	if err := s.convRepo.Update(ctx, convID, update); err != nil {
		return nil, appErr.Wrap(err, "failed to update conversation")
	}

	return s.convRepo.GetByID(ctx, convID)
}

// DeleteConversation soft-deletes a conversation with ownership verification.
func (s *ConversationService) DeleteConversation(ctx context.Context, userID, convID uuid.UUID) error {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return appErr.ErrForbidden
	}

	return s.convRepo.Delete(ctx, convID)
}

// ListMessages returns messages in a conversation with ownership verification.
func (s *ConversationService) ListMessages(ctx context.Context, userID, convID uuid.UUID, page, pageSize int) ([]*domain.Message, int, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, 0, appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return nil, 0, appErr.ErrForbidden
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	messages, total, err := s.msgRepo.ListByConversation(ctx, convID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Load attachments for each message
	if s.attRepo != nil {
		for _, msg := range messages {
			attachments, err := s.attRepo.ListByMessage(ctx, msg.ID)
			if err == nil && len(attachments) > 0 {
				msg.Attachments = make([]domain.Attachment, len(attachments))
				for i, att := range attachments {
					msg.Attachments[i] = *att
				}
			}
		}
	}

	return messages, total, nil
}
