package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	gohttp "net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/storage"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/domain"
	"github.com/omnidev/gateway/internal/repository"
)

// GenerateImageInput defines the input for image generation.
type GenerateImageInput struct {
	ConversationID string `json:"conversation_id"`
	Model          string `json:"model" validate:"required"`
	Prompt         string `json:"prompt" validate:"required"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	N              int    `json:"n,omitempty"`
	Watermark      *bool  `json:"watermark_enabled,omitempty"`
}

// GenerateImageResult represents the result of image generation.
type GenerateImageResult struct {
	Attachment    *domain.Attachment `json:"attachment"`
	RevisedPrompt string             `json:"revised_prompt,omitempty"`
}

// ImageService handles image generation operations.
type ImageService struct {
	adapterResolver *AdapterResolver
	attRepo         repository.AttachmentRepository
	convRepo        repository.ConversationRepository
	msgRepo         repository.MessageRepository
	minioCli        *storage.MinIO
}

// NewImageService creates a new image service.
func NewImageService(
	adapterResolver *AdapterResolver,
	attRepo repository.AttachmentRepository,
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	minioCli *storage.MinIO,
) *ImageService {
	return &ImageService{
		adapterResolver: adapterResolver,
		attRepo:         attRepo,
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		minioCli:        minioCli,
	}
}

// GenerateImage generates images using an AI model and stores them in MinIO.
func (s *ImageService) GenerateImage(ctx context.Context, userID uuid.UUID, input *GenerateImageInput) ([]*GenerateImageResult, error) {
	// Resolve adapter
	aiAdapter, err := s.adapterResolver.Resolve(ctx, userID, input.Model)
	if err != nil {
		return nil, appErr.Validation("unsupported model: " + input.Model)
	}

	// Check if adapter supports image generation
	imgGen, ok := aiAdapter.(adapter.ImageGenerator)
	if !ok {
		return nil, appErr.Validation("model does not support image generation: " + input.Model)
	}

	// Set defaults
	n := input.N
	if n <= 0 {
		n = 1
	}
	size := input.Size
	if size == "" {
		size = "1024x1024"
	}

	// Default watermark to false
	watermark := false
	if input.Watermark != nil {
		watermark = *input.Watermark
	}

	// Call image generation API
	resp, err := imgGen.GenerateImage(ctx, &adapter.ImageRequest{
		Model:     input.Model,
		Prompt:    input.Prompt,
		N:         n,
		Size:      size,
		Quality:   input.Quality,
		Style:     input.Style,
		Watermark: &watermark,
	})
	if err != nil {
		logger.Log.Error("Image generation failed", zap.Error(err), zap.String("model", input.Model))
		return nil, appErr.Wrap(err, "image generation failed")
	}

	if len(resp.Images) == 0 {
		return nil, appErr.New(500, "no images returned from generation")
	}

	// Store each generated image
	var results []*GenerateImageResult
	for _, img := range resp.Images {
		var imageData []byte

		if img.Base64 != "" {
			imageData, err = base64.StdEncoding.DecodeString(img.Base64)
			if err != nil {
				logger.Log.Warn("Failed to decode base64 image", zap.Error(err))
				continue
			}
		} else if img.URL != "" {
			httpResp, err := gohttp.Get(img.URL)
			if err != nil {
				logger.Log.Warn("Failed to download generated image", zap.String("url", img.URL), zap.Error(err))
				continue
			}
			imageData, err = io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			if err != nil {
				logger.Log.Warn("Failed to read image data", zap.Error(err))
				continue
			}
		} else {
			continue
		}

		if s.minioCli == nil {
			return nil, appErr.New(500, "storage not configured")
		}

		filename := fmt.Sprintf("generated_%s.png", uuid.New().String()[:8])
		objectKey := fmt.Sprintf("images/%s/%s", userID.String(), filename)

		// Upload to MinIO
		if _, err := s.minioCli.UploadBytes(ctx, "chat", objectKey, imageData, "image/png"); err != nil {
			logger.Log.Warn("Failed to upload image to storage", zap.Error(err))
			continue
		}

		// Generate presigned URL (7 days)
		presignedURL, err := s.minioCli.GetPresignedURL(ctx, "chat", objectKey, 7*24*time.Hour)
		if err != nil {
			logger.Log.Warn("Failed to generate presigned URL", zap.Error(err))
			continue
		}

		// Save attachment record
		att := &domain.Attachment{
			ID:         uuid.New(),
			UserID:     userID,
			Filename:   filename,
			MimeType:   "image/png",
			FileSize:   int64(len(imageData)),
			StorageKey: objectKey,
			StorageURL: presignedURL,
			Metadata:   map[string]interface{}{"source": "ai-generated", "model": input.Model},
		}

		if err := s.attRepo.Create(ctx, att); err != nil {
			logger.Log.Warn("Failed to save attachment record", zap.Error(err))
		}

		results = append(results, &GenerateImageResult{
			Attachment:    att,
			RevisedPrompt: img.RevisedPrompt,
		})
	}

	if len(results) == 0 {
		return nil, appErr.New(500, "failed to store any generated images")
	}

	// Persist messages to conversation if conversationID provided
	if input.ConversationID != "" {
		s.persistToConversation(ctx, userID, input, results)
	}

	return results, nil
}

// persistToConversation saves image generation messages to a conversation.
func (s *ImageService) persistToConversation(ctx context.Context, userID uuid.UUID, input *GenerateImageInput, results []*GenerateImageResult) {
	convID, err := uuid.Parse(input.ConversationID)
	if err != nil {
		return
	}

	// Save user message
	userMsg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           domain.MessageRoleUser,
		Content:        fmt.Sprintf("🎨 %s", input.Prompt),
		Metadata:       map[string]interface{}{"type": "image-generation"},
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		logger.Log.Warn("Failed to save user message for image generation", zap.Error(err))
	}

	// Save assistant message with attachments
	assistantMsg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           domain.MessageRoleAssistant,
		Content:        input.Prompt,
		ModelID:        &input.Model,
		Metadata:       map[string]interface{}{"type": "image-generation", "model": input.Model},
	}
	if err := s.msgRepo.Create(ctx, assistantMsg); err != nil {
		logger.Log.Warn("Failed to save assistant message for image generation", zap.Error(err))
	}

	// Link attachments to assistant message
	attIDs := make([]uuid.UUID, 0, len(results))
	for _, r := range results {
		attIDs = append(attIDs, r.Attachment.ID)
	}
	if len(attIDs) > 0 {
		if err := s.attRepo.UpdateMessageID(ctx, attIDs, assistantMsg.ID); err != nil {
			logger.Log.Warn("Failed to link attachments to message", zap.Error(err))
		}
	}

	// Update conversation
	_ = s.convRepo.IncrementMessageCount(ctx, convID) // user msg
	_ = s.convRepo.IncrementMessageCount(ctx, convID) // assistant msg

	// Auto-generate title
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err == nil && (conv.Title == nil || *conv.Title == "") {
		title := fmt.Sprintf("🎨 %s", generateTitle(input.Prompt))
		_ = s.convRepo.Update(ctx, convID, &repository.ConversationUpdate{Title: &title})
	}
}
