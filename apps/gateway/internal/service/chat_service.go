// Package service contains the business logic for the Gateway.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/cache"
	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/domain"
	"github.com/omnidev/gateway/internal/repository"
)

// ChatService handles chat operations.
type ChatService struct {
	convRepo      repository.ConversationRepository
	msgRepo       repository.MessageRepository
	modelRepo     repository.ModelRepository
	adapters      *adapter.Registry
	cache         *cache.Redis
	defaultModel  string
}

// NewChatService creates a new chat service.
func NewChatService(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	modelRepo repository.ModelRepository,
	adapters *adapter.Registry,
	cache *cache.Redis,
	defaultModel string,
) *ChatService {
	return &ChatService{
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		modelRepo:    modelRepo,
		adapters:     adapters,
		cache:        cache,
		defaultModel: defaultModel,
	}
}

// CreateConversationInput defines the input for creating a conversation.
type CreateConversationInput struct {
	Title        string                 `json:"title"`
	ModelID      string                 `json:"model_id"`
	SystemPrompt string                 `json:"system_prompt"`
	Tags         []string               `json:"tags"`
	Settings     map[string]interface{} `json:"settings"`
}

// CreateConversation creates a new conversation.
func (s *ChatService) CreateConversation(ctx context.Context, userID uuid.UUID, input *CreateConversationInput) (*domain.Conversation, error) {
	conv := &domain.Conversation{
		ID:       uuid.New(),
		UserID:   userID,
		Status:   domain.ConversationStatusActive,
		Tags:     input.Tags,
		Settings: input.Settings,
		Metadata: map[string]interface{}{},
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

	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, appErr.Wrap(err, "failed to create conversation")
	}

	return conv, nil
}

// GetConversation returns a conversation by ID.
func (s *ChatService) GetConversation(ctx context.Context, userID, convID uuid.UUID) (*domain.Conversation, error) {
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
func (s *ChatService) ListConversations(ctx context.Context, userID uuid.UUID, input *ListConversationsInput) ([]*domain.Conversation, int, error) {
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
	Title        *string `json:"title"`
	SystemPrompt *string `json:"system_prompt"`
	Status       *string `json:"status"`
	Pinned       *bool   `json:"pinned"`
}

// UpdateConversation updates a conversation.
func (s *ChatService) UpdateConversation(ctx context.Context, userID, convID uuid.UUID, input *UpdateConversationInput) (*domain.Conversation, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	update := &repository.ConversationUpdate{
		Title:        input.Title,
		SystemPrompt: input.SystemPrompt,
		Pinned:       input.Pinned,
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

// DeleteConversation soft-deletes a conversation.
func (s *ChatService) DeleteConversation(ctx context.Context, userID, convID uuid.UUID) error {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return appErr.ErrForbidden
	}

	return s.convRepo.Delete(ctx, convID)
}

// ListMessages returns messages in a conversation.
func (s *ChatService) ListMessages(ctx context.Context, userID, convID uuid.UUID, page, pageSize int) ([]*domain.Message, int, error) {
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
	return s.msgRepo.ListByConversation(ctx, convID, offset, pageSize)
}

// SendMessageInput defines the input for sending a message.
type SendMessageInput struct {
	Content string `json:"content" validate:"required"`
	ModelID string `json:"model_id"`
}

// SendMessage sends a message and returns the AI response.
func (s *ChatService) SendMessage(ctx context.Context, userID, convID uuid.UUID, input *SendMessageInput) (*domain.Message, *domain.Message, error) {
	// Verify ownership
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, nil, appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return nil, nil, appErr.ErrForbidden
	}

	// Determine model
	modelID := input.ModelID
	if modelID == "" && conv.ModelID != nil {
		modelID = conv.ModelID.String()
	}
	if modelID == "" {
		modelID = s.defaultModel
	}

	// Save user message
	userMsg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           domain.MessageRoleUser,
		Content:        input.Content,
		Metadata:       map[string]interface{}{},
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, nil, appErr.Wrap(err, "failed to save user message")
	}

	// Get recent messages for context
	recentMsgs, err := s.msgRepo.GetRecentMessages(ctx, convID, 20)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get context")
	}

	// Build adapter messages
	adapterMsgs := make([]adapter.Message, 0, len(recentMsgs)+1)
	if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
		adapterMsgs = append(adapterMsgs, adapter.Message{
			Role:    "system",
			Content: *conv.SystemPrompt,
		})
	}
	for _, msg := range recentMsgs {
		adapterMsgs = append(adapterMsgs, adapter.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Get adapter
	aiAdapter, err := s.adapters.GetForModel(modelID)
	if err != nil {
		return nil, nil, appErr.Validation("unsupported model: " + modelID)
	}

	// Call AI
	start := time.Now()
	resp, err := aiAdapter.Chat(ctx, &adapter.ChatRequest{
		Model:    modelID,
		Messages: adapterMsgs,
	})
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		logger.Log.Error("AI call failed", zap.Error(err), zap.String("model", modelID))
		return nil, nil, appErr.Wrap(err, "AI request failed")
	}

	// Save assistant message
	assistantMsg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           domain.MessageRoleAssistant,
		Content:        resp.Content,
		TokenInput:     &resp.Usage.PromptTokens,
		TokenOutput:    &resp.Usage.CompletionTokens,
		LatencyMs:      &latency,
		Metadata:       map[string]interface{}{"model": modelID},
	}
	if err := s.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, nil, appErr.Wrap(err, "failed to save assistant message")
	}

	// Increment message count (user + assistant)
	_ = s.convRepo.IncrementMessageCount(ctx, convID)
	_ = s.convRepo.IncrementMessageCount(ctx, convID)

	// Auto-generate title from first message
	if conv.Title == nil || *conv.Title == "" {
		title := generateTitle(input.Content)
		_ = s.convRepo.Update(ctx, convID, &repository.ConversationUpdate{Title: &title})
	}

	// Emit usage event for billing
	s.emitUsageEvent(ctx, userID, modelID, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return userMsg, assistantMsg, nil
}

// StreamMessage sends a message and streams the response.
func (s *ChatService) StreamMessage(ctx context.Context, userID, convID uuid.UUID, input *SendMessageInput) (<-chan domain.ChatChunk, *domain.Message, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil {
		return nil, nil, appErr.NotFound("conversation")
	}
	if conv.UserID != userID {
		return nil, nil, appErr.ErrForbidden
	}

	modelID := input.ModelID
	if modelID == "" && conv.ModelID != nil {
		modelID = conv.ModelID.String()
	}
	if modelID == "" {
		modelID = "gpt-4o-mini"
	}

	// Save user message
	userMsg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           domain.MessageRoleUser,
		Content:        input.Content,
		Metadata:       map[string]interface{}{},
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, nil, appErr.Wrap(err, "failed to save user message")
	}

	// Get context
	recentMsgs, err := s.msgRepo.GetRecentMessages(ctx, convID, 20)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get context")
	}

	adapterMsgs := make([]adapter.Message, 0, len(recentMsgs)+1)
	if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
		adapterMsgs = append(adapterMsgs, adapter.Message{
			Role:    "system",
			Content: *conv.SystemPrompt,
		})
	}
	for _, msg := range recentMsgs {
		adapterMsgs = append(adapterMsgs, adapter.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	aiAdapter, err := s.adapters.GetForModel(modelID)
	if err != nil {
		return nil, nil, appErr.Validation("unsupported model: " + modelID)
	}

	stream, err := aiAdapter.ChatStream(ctx, &adapter.ChatRequest{
		Model:    modelID,
		Messages: adapterMsgs,
		Stream:   true,
	})
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to start stream")
	}

	// Convert adapter stream to domain stream
	ch := make(chan domain.ChatChunk, 100)
	go func() {
		defer close(ch)

		var fullContent string
		start := time.Now()

		for chunk := range stream {
			fullContent += chunk.Delta
			ch <- domain.ChatChunk{
				ID:      chunk.ID,
				Delta:   chunk.Delta,
				ModelID: modelID,
			}

			if chunk.Finish == "stop" {
				latency := int(time.Since(start).Milliseconds())

				// Save complete assistant message
				assistantMsg := &domain.Message{
					ID:             uuid.New(),
					ConversationID: convID,
					Role:           domain.MessageRoleAssistant,
					Content:        fullContent,
					LatencyMs:      &latency,
					Metadata:       map[string]interface{}{"model": modelID, "streamed": true},
				}
				_ = s.msgRepo.Create(ctx, assistantMsg)
				_ = s.convRepo.IncrementMessageCount(ctx, convID)
				_ = s.convRepo.IncrementMessageCount(ctx, convID)

				if conv.Title == nil || *conv.Title == "" {
					title := generateTitle(input.Content)
					_ = s.convRepo.Update(ctx, convID, &repository.ConversationUpdate{Title: &title})
				}
			}
		}
	}()

	return ch, userMsg, nil
}

// ListModels returns available AI models.
func (s *ChatService) ListModels(ctx context.Context) ([]*domain.Model, error) {
	return s.modelRepo.List(ctx, true)
}

// emitUsageEvent publishes a usage event for billing.
func (s *ChatService) emitUsageEvent(ctx context.Context, userID uuid.UUID, model string, inputTokens, outputTokens int) {
	usageData := map[string]interface{}{
		"user_id":       userID.String(),
		"model":         model,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"timestamp":     time.Now().Unix(),
	}
	data, _ := json.Marshal(usageData)
	_ = s.cache.Set(ctx, fmt.Sprintf("usage:%s:%d", userID.String(), time.Now().UnixNano()), data, 24*time.Hour)
}

// generateTitle creates a short title from the first message.
func generateTitle(content string) string {
	if len(content) > 50 {
		return content[:50] + "..."
	}
	return content
}
