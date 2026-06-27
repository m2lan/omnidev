package service

import (
	"context"

	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"

	"github.com/omnidev/services/chat/internal/domain"
	"github.com/omnidev/services/chat/internal/repository"
)

// PromptService handles prompt template operations.
type PromptService struct {
	promptRepo repository.PromptRepository
}

// NewPromptService creates a new prompt service.
func NewPromptService(promptRepo repository.PromptRepository) *PromptService {
	return &PromptService{promptRepo: promptRepo}
}

// CreatePromptInput defines the input for creating a prompt.
type CreatePromptInput struct {
	Title       string                 `json:"title" validate:"required,min=1,max=255"`
	Content     string                 `json:"content" validate:"required"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Variables   []interface{}          `json:"variables"`
	Visibility  string                 `json:"visibility"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CreatePrompt creates a new prompt template.
func (s *PromptService) CreatePrompt(ctx context.Context, userID uuid.UUID, input *CreatePromptInput) (*domain.PromptTemplate, error) {
	visibility := input.Visibility
	if visibility == "" {
		visibility = "private"
	}

	prompt := &domain.PromptTemplate{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       input.Title,
		Content:     input.Content,
		Description: strPtr(input.Description),
		Category:    strPtr(input.Category),
		Tags:        input.Tags,
		Variables:   input.Variables,
		Visibility:  visibility,
		Metadata:    input.Metadata,
	}

	if prompt.Tags == nil {
		prompt.Tags = []string{}
	}
	if prompt.Variables == nil {
		prompt.Variables = []interface{}{}
	}
	if prompt.Metadata == nil {
		prompt.Metadata = map[string]interface{}{}
	}

	if err := s.promptRepo.Create(ctx, prompt); err != nil {
		return nil, appErr.Wrap(err, "failed to create prompt")
	}

	return prompt, nil
}

// GetPrompt returns a prompt by ID.
func (s *PromptService) GetPrompt(ctx context.Context, userID, promptID uuid.UUID) (*domain.PromptTemplate, error) {
	prompt, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return nil, appErr.NotFound("prompt")
	}

	// Check access: owner or public
	if prompt.UserID != userID && prompt.Visibility != "public" {
		return nil, appErr.ErrForbidden
	}

	return prompt, nil
}

// ListPromptsInput defines filters for listing prompts.
type ListPromptsInput struct {
	Visibility string `form:"visibility"`
	Category   string `form:"category"`
	Search     string `form:"search"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// ListPrompts returns a paginated list of prompts.
func (s *PromptService) ListPrompts(ctx context.Context, userID uuid.UUID, input *ListPromptsInput) ([]*domain.PromptTemplate, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	filter := &repository.PromptFilter{
		Search: input.Search,
	}
	if input.Visibility != "" {
		filter.Visibility = &input.Visibility
	}
	if input.Category != "" {
		filter.Category = &input.Category
	}

	offset := (input.Page - 1) * input.PageSize
	return s.promptRepo.List(ctx, userID, filter, offset, input.PageSize)
}

// UpdatePromptInput defines fields for updating a prompt.
type UpdatePromptInput struct {
	Title       *string  `json:"title"`
	Content     *string  `json:"content"`
	Description *string  `json:"description"`
	Category    *string  `json:"category"`
	Tags        []string `json:"tags"`
	Visibility  *string  `json:"visibility"`
}

// UpdatePrompt updates a prompt template.
func (s *PromptService) UpdatePrompt(ctx context.Context, userID, promptID uuid.UUID, input *UpdatePromptInput) (*domain.PromptTemplate, error) {
	prompt, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return nil, appErr.NotFound("prompt")
	}
	if prompt.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	update := &repository.PromptUpdate{
		Title:       input.Title,
		Content:     input.Content,
		Description: input.Description,
		Category:    input.Category,
		Tags:        input.Tags,
		Visibility:  input.Visibility,
	}

	if err := s.promptRepo.Update(ctx, promptID, update); err != nil {
		return nil, appErr.Wrap(err, "failed to update prompt")
	}

	return s.promptRepo.GetByID(ctx, promptID)
}

// DeletePrompt soft-deletes a prompt template.
func (s *PromptService) DeletePrompt(ctx context.Context, userID, promptID uuid.UUID) error {
	prompt, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return appErr.NotFound("prompt")
	}
	if prompt.UserID != userID {
		return appErr.ErrForbidden
	}

	return s.promptRepo.Delete(ctx, promptID)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
