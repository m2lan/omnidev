// Package handler provides HTTP handlers for the Gateway.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/service"
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	// Parse multipart form (max 20MB)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": "failed to parse form: " + err.Error()}})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": "file is required"}})
		return
	}
	defer file.Close()

	if header.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": "filename is required"}})
		return
	}

	att, err := h.uploadSvc.UploadFile(c.Request.Context(), userID, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "upload failed", "detail": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": att})
}

// GetAttachment returns attachment metadata.
// GET /api/v1/attachments/:id
func (h *UploadHandler) GetAttachment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	attID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": "invalid attachment ID"}})
		return
	}

	att, err := h.uploadSvc.GetAttachment(c.Request.Context(), userID, attID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": 404, "message": "not found", "detail": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": att})
}

// DeleteAttachment deletes an attachment.
// DELETE /api/v1/attachments/:id
func (h *UploadHandler) DeleteAttachment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	attID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": "invalid attachment ID"}})
		return
	}

	if err := h.uploadSvc.DeleteAttachment(c.Request.Context(), userID, attID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": 404, "message": "not found", "detail": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "attachment deleted"}})
}
