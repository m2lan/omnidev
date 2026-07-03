// Package handler provides HTTP handlers for the Chat Service.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/chat/internal/service"
)

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	uploadSvc *service.UploadService
}

// NewUploadHandler creates a new upload handler.
func NewUploadHandler(uploadSvc *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadSvc: uploadSvc}
}

// Upload uploads a file and returns attachment metadata.
// POST /api/v1/upload
func (h *UploadHandler) Upload(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	// Parse multipart form (max 20MB)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		badRequest(c, "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	// Set filename from header if not present
	if header.Filename == "" {
		badRequest(c, "filename is required")
		return
	}

	att, err := h.uploadSvc.UploadFile(c.Request.Context(), userID, header)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": att})
}

// GetAttachment returns attachment metadata.
// GET /api/v1/attachments/:id
func (h *UploadHandler) GetAttachment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	attID := c.Param("id")
	if attID == "" {
		badRequest(c, "attachment ID is required")
		return
	}

	att, err := h.uploadSvc.GetAttachment(c.Request.Context(), userID, attID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": att})
}

// GetPresignedURL returns a fresh presigned URL for an attachment.
// GET /api/v1/attachments/:id/url
func (h *UploadHandler) GetPresignedURL(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	attID := c.Param("id")
	if attID == "" {
		badRequest(c, "attachment ID is required")
		return
	}

	url, err := h.uploadSvc.GetPresignedURL(c.Request.Context(), userID, attID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"url": url}})
}

// DeleteAttachment deletes an attachment.
// DELETE /api/v1/attachments/:id
func (h *UploadHandler) DeleteAttachment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	attID := c.Param("id")
	if attID == "" {
		badRequest(c, "attachment ID is required")
		return
	}

	if err := h.uploadSvc.DeleteAttachment(c.Request.Context(), userID, attID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "attachment deleted"}})
}
