// Package service contains the business logic for the Gateway.
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/cache"
	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/parser"
	"github.com/omnidev/go-common/storage"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/domain"
	"github.com/omnidev/gateway/internal/repository"
	ragservice "github.com/omnidev/gateway/internal/rag/service"
)

// ChatService handles message sending and streaming operations.
type ChatService struct {
	convRepo        repository.ConversationRepository
	msgRepo         repository.MessageRepository
	modelRepo       repository.ModelRepository
	attRepo         repository.AttachmentRepository
	minioCli        *storage.MinIO
	parser          parser.Parser
	adapters        *adapter.Registry
	adapterResolver *AdapterResolver
	userConfigRepo  repository.UserAIConfigRepository
	cache           *cache.Redis
	defaultModel    string
	ragService      *RAGService
	userRepo        repository.UserRepository
	kbService       *ragservice.KnowledgeBaseService
}

// NewChatService creates a new chat service.
func NewChatService(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	modelRepo repository.ModelRepository,
	adapters *adapter.Registry,
	adapterResolver *AdapterResolver,
	userConfigRepo repository.UserAIConfigRepository,
	cache *cache.Redis,
	defaultModel string,
	opts ...interface{},
) *ChatService {
	svc := &ChatService{
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		modelRepo:       modelRepo,
		adapters:        adapters,
		adapterResolver: adapterResolver,
		userConfigRepo:  userConfigRepo,
		cache:           cache,
		defaultModel:    defaultModel,
	}
	for _, opt := range opts {
		switch v := opt.(type) {
		case repository.AttachmentRepository:
			svc.attRepo = v
		case *storage.MinIO:
			svc.minioCli = v
		case parser.Parser:
			svc.parser = v
		case *RAGService:
			svc.ragService = v
		case repository.UserRepository:
			svc.userRepo = v
		case *ragservice.KnowledgeBaseService:
			svc.kbService = v
		}
	}
	return svc
}

// SendMessageInput defines the input for sending a message.
type SendMessageInput struct {
	Content          string   `json:"content" validate:"required"`
	ModelID          string   `json:"model_id"`
	AttachmentIDs    []string `json:"attachment_ids,omitempty"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
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

	// Associate attachments with message
	if len(input.AttachmentIDs) > 0 && s.attRepo != nil {
		attUUIDs := make([]uuid.UUID, 0, len(input.AttachmentIDs))
		for _, idStr := range input.AttachmentIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				attUUIDs = append(attUUIDs, id)
			}
		}
		if len(attUUIDs) > 0 {
			if err := s.attRepo.UpdateMessageID(ctx, attUUIDs, userMsg.ID); err != nil {
				logger.Log.Warn("failed to associate attachments", zap.Error(err))
			}
		}
	}

	// Get recent messages for context
	recentMsgs, err := s.msgRepo.GetRecentMessages(ctx, convID, 20)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get context")
	}

	// Build adapter messages
	adapterMsgs := s.buildAdapterMessages(ctx, conv, recentMsgs, input)

	// Get adapter via resolver
	aiAdapter, err := s.adapterResolver.Resolve(ctx, userID, modelID)
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
		ModelID:        &modelID,
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

	// Associate attachments with message
	if len(input.AttachmentIDs) > 0 && s.attRepo != nil {
		attUUIDs := make([]uuid.UUID, 0, len(input.AttachmentIDs))
		for _, idStr := range input.AttachmentIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				attUUIDs = append(attUUIDs, id)
			}
		}
		if len(attUUIDs) > 0 {
			if err := s.attRepo.UpdateMessageID(ctx, attUUIDs, userMsg.ID); err != nil {
				logger.Log.Warn("failed to associate attachments", zap.Error(err))
			}
		}
	}

	// Get context
	recentMsgs, err := s.msgRepo.GetRecentMessages(ctx, convID, 20)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get context")
	}

	// Build adapter messages
	adapterMsgs := s.buildAdapterMessages(ctx, conv, recentMsgs, input)

	// Get adapter via resolver
	aiAdapter, err := s.adapterResolver.Resolve(ctx, userID, modelID)
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
				ID:           chunk.ID,
				Delta:        chunk.Delta,
				Type:         chunk.Type,
				FinishReason: chunk.Finish,
				ModelID:      modelID,
			}

			if chunk.Finish == "stop" {
				latency := int(time.Since(start).Milliseconds())

				// Estimate token counts
				inputTokens := estimateTokens(adapterMsgs)
				outputTokens := estimateTokensString(fullContent)

				// Save complete assistant message
				assistantMsg := &domain.Message{
					ID:             uuid.New(),
					ConversationID: convID,
					Role:           domain.MessageRoleAssistant,
					Content:        fullContent,
					ModelID:        &modelID,
					TokenInput:     &inputTokens,
					TokenOutput:    &outputTokens,
					LatencyMs:      &latency,
					Metadata:       map[string]interface{}{"model": modelID, "streamed": true},
				}
				_ = s.msgRepo.Create(ctx, assistantMsg)
				_ = s.convRepo.IncrementMessageCount(ctx, convID)

				if conv.Title == nil || *conv.Title == "" {
					title := generateTitle(input.Content)
					_ = s.convRepo.Update(ctx, convID, &repository.ConversationUpdate{Title: &title})
				}

				// Send completion event with full message
				ch <- domain.ChatChunk{
					ID:           assistantMsg.ID.String(),
					FinishReason: "stop",
					TokenInput:   inputTokens,
					TokenOutput:  outputTokens,
					ModelID:      modelID,
				}
			}
		}
	}()

	return ch, userMsg, nil
}

// ListModels returns available AI models from database.
func (s *ChatService) ListModels(ctx context.Context) ([]*domain.Model, error) {
	return s.modelRepo.List(ctx, true)
}

// ListAvailableModels returns models from registered adapters.
func (s *ChatService) ListAvailableModels() []map[string]interface{} {
	var models []map[string]interface{}
	id := 0
	for _, provider := range s.adapters.Providers() {
		adp, err := s.adapters.Get(provider)
		if err != nil {
			continue
		}
		for _, modelID := range adp.Models() {
			id++
			models = append(models, map[string]interface{}{
				"id":           fmt.Sprintf("%d", id),
				"provider":     provider,
				"model_id":     modelID,
				"display_name": modelID,
			})
		}
	}
	return models
}

// ListAvailableModelsForUser returns models from global adapters and user configs.
func (s *ChatService) ListAvailableModelsForUser(ctx context.Context, userID uuid.UUID) []map[string]interface{} {
	var models []map[string]interface{}
	id := 0
	seen := make(map[string]bool)

	// Add user config models first (higher priority)
	if s.userConfigRepo != nil {
		userConfigs, err := s.userConfigRepo.ListByUserID(ctx, userID)
		if err == nil {
			for _, cfg := range userConfigs {
				if cfg.Provider == "" {
					continue
				}
				for _, modelID := range cfg.Models {
					key := cfg.Provider + ":" + modelID
					if seen[key] {
						continue
					}
					seen[key] = true
					id++
					models = append(models, map[string]interface{}{
						"id":           fmt.Sprintf("%d", id),
						"provider":     cfg.Provider,
						"model_id":     modelID,
						"display_name": modelID,
						"source":       "user",
					})
				}
			}
		}
	}

	// Add global adapter models (fallback)
	for _, provider := range s.adapters.Providers() {
		adp, err := s.adapters.Get(provider)
		if err != nil {
			continue
		}
		for _, modelID := range adp.Models() {
			key := provider + ":" + modelID
			if seen[key] {
				continue
			}
			seen[key] = true
			id++
			models = append(models, map[string]interface{}{
				"id":           fmt.Sprintf("%d", id),
				"provider":     provider,
				"model_id":     modelID,
				"display_name": modelID,
				"source":       "global",
			})
		}
	}

	return models
}

// resolveKnowledgeBaseIDs determines which knowledge bases to search based on the 3-tier priority:
// 1. Message-level override (frontend temporary selection) — highest priority
// 2. Conversation-level binding (set at conversation creation) — persisted in DB
// 3. User default (configured in user settings) — lowest priority
// Returns the resolved KB IDs, or nil if RAG is disabled.
func (s *ChatService) resolveKnowledgeBaseIDs(
	ctx context.Context,
	userID uuid.UUID,
	messageKBIDs []string,
	conversationKBIDs []string,
) []string {
	// 1. Message-level override
	if len(messageKBIDs) > 0 {
		return messageKBIDs
	}

	// Fetch user settings if userRepo is available
	if s.userRepo != nil {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err == nil && user.Settings != nil {
			ragMode, _ := user.Settings["rag_mode"].(string)

			switch ragMode {
			case "off":
				return nil
			case "all":
				// Return sentinel for caller to resolve
				return []string{"__all__"}
			case "specified":
				// Fall through to conversation/user default logic
			default:
				// No rag_mode set — check conversation and user defaults
			}

			// 2. Conversation-level binding
			if len(conversationKBIDs) > 0 {
				return conversationKBIDs
			}

			// 3. User default KB IDs
			if defaultIDs, ok := user.Settings["default_kb_ids"].([]interface{}); ok {
				ids := make([]string, 0, len(defaultIDs))
				for _, id := range defaultIDs {
					if s, ok := id.(string); ok {
						ids = append(ids, s)
					}
				}
				if len(ids) > 0 {
					return ids
				}
			}
		}
	}

	// Fallback: use conversation-level binding if available
	if len(conversationKBIDs) > 0 {
		return conversationKBIDs
	}

	return nil
}

// buildAdapterMessages converts domain messages to adapter messages with multimodal support.
func (s *ChatService) buildAdapterMessages(ctx context.Context, conv *domain.Conversation, recentMsgs []*domain.Message, input *SendMessageInput) []adapter.Message {
	adapterMsgs := make([]adapter.Message, 0, len(recentMsgs)+2)
	if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
		adapterMsgs = append(adapterMsgs, adapter.NewTextMessage("system", *conv.SystemPrompt))
	}

	// Inject RAG context — resolve KB IDs using 3-tier priority
	if s.ragService != nil {
		convKBStrs := make([]string, len(conv.KnowledgeBaseIDs))
		for i, id := range conv.KnowledgeBaseIDs {
			convKBStrs[i] = id.String()
		}
		resolvedIDs := s.resolveKnowledgeBaseIDs(ctx, conv.UserID, input.KnowledgeBaseIDs, convKBStrs)

		// Handle "__all__" sentinel — fetch all user's knowledge bases
		if len(resolvedIDs) == 1 && resolvedIDs[0] == "__all__" && s.kbService != nil {
			allKBs, err := s.kbService.ListAllKnowledgeBases(ctx, conv.UserID)
			if err == nil && len(allKBs) > 0 {
				resolvedIDs = make([]string, len(allKBs))
				for i, kb := range allKBs {
					resolvedIDs[i] = kb.ID.String()
				}
			} else {
				resolvedIDs = nil
			}
		}

		if len(resolvedIDs) > 0 {
			ragContext := s.retrieveRAGContext(ctx, input.Content, resolvedIDs)
			if ragContext != "" {
				adapterMsgs = append(adapterMsgs, adapter.NewTextMessage("system",
					"The following context was retrieved from the user's knowledge bases. Use it to answer the user's question when relevant. Cite the source when using information from the knowledge base.\n\n"+ragContext))
			}
		}
	}

	for _, msg := range recentMsgs {
		var imageDataURLs []string
		var docContents []string
		if s.attRepo != nil {
			attachments, err := s.attRepo.ListByMessage(ctx, msg.ID)
			if err == nil {
				for _, att := range attachments {
					if att.IsImage() {
						dataURL, err := s.imageToBase64(ctx, att)
						if err == nil {
							imageDataURLs = append(imageDataURLs, dataURL)
						}
					} else if att.IsDocument() {
						text, err := s.downloadAndParse(ctx, att)
						if err == nil {
							docContents = append(docContents, fmt.Sprintf("\n\n[附件: %s]\n%s", att.Filename, text))
						}
					}
				}
			}
		}

		content := msg.Content
		if len(docContents) > 0 {
			content += strings.Join(docContents, "")
		}

		if len(imageDataURLs) > 0 {
			adapterMsgs = append(adapterMsgs, adapter.NewMultimodalMessage(string(msg.Role), content, imageDataURLs))
		} else {
			adapterMsgs = append(adapterMsgs, adapter.NewTextMessage(string(msg.Role), content))
		}
	}

	// Also check current message attachments
	var currentImageDataURLs []string
	var currentDocContents []string
	if len(input.AttachmentIDs) > 0 && s.attRepo != nil {
		for _, idStr := range input.AttachmentIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				att, err := s.attRepo.GetByID(ctx, id)
				if err == nil {
					if att.IsImage() {
						dataURL, err := s.imageToBase64(ctx, att)
						if err == nil {
							currentImageDataURLs = append(currentImageDataURLs, dataURL)
						}
					} else if att.IsDocument() {
						text, err := s.downloadAndParse(ctx, att)
						if err == nil {
							currentDocContents = append(currentDocContents, fmt.Sprintf("\n\n[附件: %s]\n%s", att.Filename, text))
						}
					}
				}
			}
		}
	}

	// Build final content for current message
	currentContent := input.Content
	if len(currentDocContents) > 0 {
		currentContent += strings.Join(currentDocContents, "")
	}

	// Update the last user message in adapterMsgs
	if (len(currentImageDataURLs) > 0 || len(currentDocContents) > 0) && len(adapterMsgs) > 0 {
		lastIdx := len(adapterMsgs) - 1
		if adapterMsgs[lastIdx].Role == "user" {
			if len(currentImageDataURLs) > 0 {
				adapterMsgs[lastIdx] = adapter.NewMultimodalMessage("user", currentContent, currentImageDataURLs)
			} else {
				adapterMsgs[lastIdx] = adapter.NewTextMessage("user", currentContent)
			}
		}
	}

	return adapterMsgs
}

// retrieveRAGContext searches knowledge bases and formats the results as context for the LLM.
func (s *ChatService) retrieveRAGContext(ctx context.Context, query string, kbIDs []string) string {
	if s.ragService == nil {
		return ""
	}

	// Get the auth token from context (passed by handler)
	token, _ := ctx.Value("auth_token").(string)
	if token == "" {
		return ""
	}

	var allResults []RAGSearchResult
	for _, kbID := range kbIDs {
		results, err := s.ragService.Search(ctx, token, kbID, query, 3)
		if err != nil {
			logger.Log.Warn("RAG search failed for KB", zap.String("kb_id", kbID), zap.Error(err))
			continue
		}
		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		return ""
	}

	// Format context
	var sb strings.Builder
	sb.WriteString("---\n")
	for i, r := range allResults {
		if i >= 10 { // Limit to top 10 results total
			break
		}
		sb.WriteString(fmt.Sprintf("[Source %d | Score: %.3f]\n%s\n\n", i+1, r.Score, r.Chunk.Content))
	}
	sb.WriteString("---")
	return sb.String()
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

// estimateTokens estimates token count from adapter messages.
func estimateTokens(msgs []adapter.Message) int {
	total := 0
	for _, msg := range msgs {
		total += adapter.GetContentLength(msg) + len(msg.Role) + 4 // overhead
	}
	// Rough: ~4 chars per token
	return total / 4
}

// estimateTokensString estimates token count from a string.
func estimateTokensString(content string) int {
	return len(content) / 4
}

// imageToBase64 downloads an image from MinIO and returns a data URL.
func (s *ChatService) imageToBase64(ctx context.Context, att *domain.Attachment) (string, error) {
	if s.minioCli == nil {
		return att.StorageURL, nil // fallback to URL
	}

	reader, err := s.minioCli.Download(ctx, "chat", att.StorageKey)
	if err != nil {
		return att.StorageURL, nil // fallback
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return att.StorageURL, nil
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", att.MimeType, encoded), nil
}

// downloadAttachment downloads an attachment from MinIO.
func (s *ChatService) downloadAttachment(ctx context.Context, att *domain.Attachment) ([]byte, error) {
	if s.minioCli == nil {
		return nil, fmt.Errorf("minio not configured")
	}

	reader, err := s.minioCli.Download(ctx, "chat", att.StorageKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// downloadAndParse downloads a document and extracts text using the parser.
func (s *ChatService) downloadAndParse(ctx context.Context, att *domain.Attachment) (string, error) {
	if s.minioCli == nil {
		return "", fmt.Errorf("minio not configured")
	}

	// If no parser configured, fall back to raw bytes
	if s.parser == nil {
		logger.Log.Warn("No parser configured, falling back to raw bytes",
			zap.String("filename", att.Filename),
		)
		data, err := s.downloadAttachment(ctx, att)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Download file from MinIO
	reader, err := s.minioCli.Download(ctx, "chat", att.StorageKey)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	logger.Log.Info("Parsing document with Tika",
		zap.String("filename", att.Filename),
		zap.String("mime_type", att.MimeType),
	)

	// Parse document using Tika
	result, err := s.parser.Parse(ctx, att.Filename, reader)
	if err != nil {
		logger.Log.Error("Failed to parse document",
			zap.String("filename", att.Filename),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to parse document: %w", err)
	}

	logger.Log.Info("Document parsed successfully",
		zap.String("filename", att.Filename),
		zap.Int("content_length", len(result.Content)),
		zap.Int("pages", result.Pages),
	)

	return result.Content, nil
}
