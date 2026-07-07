// Package handler provides HTTP handlers for the Gateway.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/service"
)

// ChatHandler handles chat endpoints.
type ChatHandler struct {
	chatSvc *service.ChatService
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

// ListConversations returns a paginated list of conversations.
// GET /api/v1/conversations
func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	input := &service.ListConversationsInput{
		Status:   c.Query("status"),
		ModelID:  c.Query("model_id"),
		Search:   c.Query("search"),
		Page:     chatQueryInt(c, "page", 1),
		PageSize: chatQueryInt(c, "page_size", 20),
	}

	convs, total, err := h.chatSvc.ListConversations(c.Request.Context(), userID, input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": convs,
		"meta": gin.H{
			"total_count": total,
			"page":        input.Page,
			"page_size":   input.PageSize,
		},
	})
}

// CreateConversation creates a new conversation.
// POST /api/v1/conversations
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	var input service.CreateConversationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		chatBadRequest(c, err.Error())
		return
	}

	conv, err := h.chatSvc.CreateConversation(c.Request.Context(), userID, &input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": conv})
}

// GetConversation returns a conversation by ID.
// GET /api/v1/conversations/:id
func (h *ChatHandler) GetConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	conv, err := h.chatSvc.GetConversation(c.Request.Context(), userID, convID)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conv})
}

// UpdateConversation updates a conversation.
// PATCH /api/v1/conversations/:id
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	var input service.UpdateConversationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		chatBadRequest(c, err.Error())
		return
	}

	conv, err := h.chatSvc.UpdateConversation(c.Request.Context(), userID, convID, &input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conv})
}

// DeleteConversation deletes a conversation.
// DELETE /api/v1/conversations/:id
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	if err := h.chatSvc.DeleteConversation(c.Request.Context(), userID, convID); err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "conversation deleted"}})
}

// ListMessages returns messages in a conversation.
// GET /api/v1/conversations/:id/messages
func (h *ChatHandler) ListMessages(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	page := chatQueryInt(c, "page", 1)
	pageSize := chatQueryInt(c, "page_size", 50)

	msgs, total, err := h.chatSvc.ListMessages(c.Request.Context(), userID, convID, page, pageSize)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": msgs,
		"meta": gin.H{
			"total_count": total,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

// SendMessage sends a message and returns the AI response.
// POST /api/v1/conversations/:id/messages
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	var input service.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		chatBadRequest(c, err.Error())
		return
	}

	// Allow empty content if there are attachments
	if input.Content == "" && len(input.AttachmentIDs) == 0 {
		chatBadRequest(c, "content or attachments required")
		return
	}
	if input.Content == "" {
		input.Content = "(file attachment)"
	}

	userMsg, assistantMsg, err := h.chatSvc.SendMessage(c.Request.Context(), userID, convID, &input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user_message":      userMsg,
			"assistant_message": assistantMsg,
		},
	})
}

// StreamMessage sends a message and streams the AI response via SSE.
// POST /api/v1/conversations/:id/messages/stream
func (h *ChatHandler) StreamMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		chatBadRequest(c, "invalid conversation ID")
		return
	}

	var input service.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		chatBadRequest(c, err.Error())
		return
	}

	// Allow empty content if there are attachments
	if input.Content == "" && len(input.AttachmentIDs) == 0 {
		chatBadRequest(c, "content or attachments required")
		return
	}
	if input.Content == "" {
		input.Content = "(file attachment)"
	}

	stream, userMsg, err := h.chatSvc.StreamMessage(c.Request.Context(), userID, convID, &input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send user message event
	userData, _ := json.Marshal(userMsg)
	fmt.Fprintf(c.Writer, "event: user_message\ndata: %s\n\n", userData)
	c.Writer.Flush()

	// Stream AI response
	var fullContent string
	completed := false
	for chunk := range stream {
		if chunk.FinishReason == "stop" && !completed {
			completed = true
			// Send complete assistant message
			completeMsg := map[string]interface{}{
				"id":              chunk.ID,
				"conversation_id": convID.String(),
				"role":            "assistant",
				"content":         fullContent,
				"model_id":        chunk.ModelID,
				"token_input":     chunk.TokenInput,
				"token_output":    chunk.TokenOutput,
				"created_at":      time.Now().UTC().Format(time.RFC3339Nano),
			}
			msgData, _ := json.Marshal(completeMsg)
			fmt.Fprintf(c.Writer, "event: complete\ndata: %s\n\n", msgData)
			c.Writer.Flush()
		} else if chunk.FinishReason != "stop" {
			fullContent += chunk.Delta
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		}
	}

	// Send done event
	fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
	c.Writer.Flush()
}

// ListModels returns available AI models.
// GET /api/v1/models
func (h *ChatHandler) ListModels(c *gin.Context) {
	// Try to get user ID for personalized model list
	if userID, ok := middleware.GetUserID(c); ok {
		models := h.chatSvc.ListAvailableModelsForUser(c.Request.Context(), userID)
		c.JSON(http.StatusOK, gin.H{"data": models})
		return
	}

	// Fallback to global models
	models := h.chatSvc.ListAvailableModels()
	c.JSON(http.StatusOK, gin.H{"data": models})
}

// GenerateImage generates images using an AI model.
// POST /api/v1/images/generate
func (h *ChatHandler) GenerateImage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		chatUnauthorized(c)
		return
	}

	var input service.GenerateImageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		chatBadRequest(c, err.Error())
		return
	}

	if input.Model == "" {
		chatBadRequest(c, "model is required")
		return
	}
	if input.Prompt == "" {
		chatBadRequest(c, "prompt is required")
		return
	}

	results, err := h.chatSvc.GenerateImage(c.Request.Context(), userID, &input)
	if err != nil {
		chatHandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

// --- Helper functions ---

func chatUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": 401, "message": "unauthorized"},
	})
}

func chatBadRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{"code": 400, "message": "bad request", "detail": detail},
	})
}

func chatHandleError(c *gin.Context, err error) {
	if e, ok := err.(*appErr.AppError); ok {
		c.JSON(e.Code, gin.H{
			"error": gin.H{
				"code":       e.Code,
				"message":    e.Message,
				"detail":     e.Detail,
				"request_id": c.GetString("X-Request-ID"),
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":       500,
			"message":    "internal server error",
			"request_id": c.GetString("X-Request-ID"),
		},
	})
}

func chatQueryInt(c *gin.Context, key string, defaultVal int) int {
	val, err := strconv.Atoi(c.DefaultQuery(key, fmt.Sprintf("%d", defaultVal)))
	if err != nil {
		return defaultVal
	}
	return val
}
