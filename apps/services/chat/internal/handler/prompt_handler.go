package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/chat/internal/service"
)

// PromptHandler handles prompt template endpoints.
type PromptHandler struct {
	promptSvc *service.PromptService
}

// NewPromptHandler creates a new prompt handler.
func NewPromptHandler(promptSvc *service.PromptService) *PromptHandler {
	return &PromptHandler{promptSvc: promptSvc}
}

// ListPrompts returns a paginated list of prompts.
// GET /api/v1/prompts
func (h *PromptHandler) ListPrompts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	input := &service.ListPromptsInput{
		Visibility: c.Query("visibility"),
		Category:   c.Query("category"),
		Search:     c.Query("search"),
		Page:       queryInt(c, "page", 1),
		PageSize:   queryInt(c, "page_size", 20),
	}

	prompts, total, err := h.promptSvc.ListPrompts(c.Request.Context(), userID, input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": prompts,
		"meta": gin.H{
			"total_count": total,
			"page":        input.Page,
			"page_size":   input.PageSize,
		},
	})
}

// CreatePrompt creates a new prompt template.
// POST /api/v1/prompts
func (h *PromptHandler) CreatePrompt(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.CreatePromptInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	prompt, err := h.promptSvc.CreatePrompt(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": prompt})
}

// GetPrompt returns a prompt by ID.
// GET /api/v1/prompts/:id
func (h *PromptHandler) GetPrompt(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	promptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid prompt ID")
		return
	}

	prompt, err := h.promptSvc.GetPrompt(c.Request.Context(), userID, promptID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": prompt})
}

// UpdatePrompt updates a prompt template.
// PATCH /api/v1/prompts/:id
func (h *PromptHandler) UpdatePrompt(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	promptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid prompt ID")
		return
	}

	var input service.UpdatePromptInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	prompt, err := h.promptSvc.UpdatePrompt(c.Request.Context(), userID, promptID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": prompt})
}

// DeletePrompt deletes a prompt template.
// DELETE /api/v1/prompts/:id
func (h *PromptHandler) DeletePrompt(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	promptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid prompt ID")
		return
	}

	if err := h.promptSvc.DeletePrompt(c.Request.Context(), userID, promptID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "prompt deleted"}})
}
