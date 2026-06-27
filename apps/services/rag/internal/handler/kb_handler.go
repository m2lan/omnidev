// Package handler provides HTTP handlers for the RAG Service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/rag/internal/service"
)

// KnowledgeBaseHandler handles knowledge base endpoints.
type KnowledgeBaseHandler struct {
	kbSvc *service.KnowledgeBaseService
}

// NewKnowledgeBaseHandler creates a new knowledge base handler.
func NewKnowledgeBaseHandler(kbSvc *service.KnowledgeBaseService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{kbSvc: kbSvc}
}

// ListKnowledgeBases returns a paginated list of knowledge bases.
// GET /api/v1/knowledge
func (h *KnowledgeBaseHandler) ListKnowledgeBases(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	kbs, total, err := h.kbSvc.ListKnowledgeBases(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": kbs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// CreateKnowledgeBase creates a new knowledge base.
// POST /api/v1/knowledge
func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.CreateKBInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	kb, err := h.kbSvc.CreateKnowledgeBase(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": kb})
}

// GetKnowledgeBase returns a knowledge base by ID.
// GET /api/v1/knowledge/:id
func (h *KnowledgeBaseHandler) GetKnowledgeBase(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	kb, err := h.kbSvc.GetKnowledgeBase(c.Request.Context(), userID, kbID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": kb})
}

// UpdateKnowledgeBase updates a knowledge base.
// PATCH /api/v1/knowledge/:id
func (h *KnowledgeBaseHandler) UpdateKnowledgeBase(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	var input service.CreateKBInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	kb, err := h.kbSvc.UpdateKnowledgeBase(c.Request.Context(), userID, kbID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": kb})
}

// DeleteKnowledgeBase deletes a knowledge base.
// DELETE /api/v1/knowledge/:id
func (h *KnowledgeBaseHandler) DeleteKnowledgeBase(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	if err := h.kbSvc.DeleteKnowledgeBase(c.Request.Context(), userID, kbID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "knowledge base deleted"}})
}

// ListDocuments returns documents in a knowledge base.
// GET /api/v1/knowledge/:id/documents
func (h *KnowledgeBaseHandler) ListDocuments(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	docs, total, err := h.kbSvc.ListDocuments(c.Request.Context(), userID, kbID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": docs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// UploadDocument uploads a document to a knowledge base.
// POST /api/v1/knowledge/:id/documents
func (h *KnowledgeBaseHandler) UploadDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	doc, err := h.kbSvc.UploadDocument(c.Request.Context(), userID, kbID, header.Filename, header.Size, file)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": doc})
}

// DeleteDocument deletes a document.
// DELETE /api/v1/knowledge/:id/documents/:doc_id
func (h *KnowledgeBaseHandler) DeleteDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid knowledge base ID")
		return
	}

	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		badRequest(c, "invalid document ID")
		return
	}

	if err := h.kbSvc.DeleteDocument(c.Request.Context(), userID, kbID, docID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "document deleted"}})
}

// --- Helper functions ---

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
}

func badRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": detail}})
}

func handleError(c *gin.Context, err error) {
	if e, ok := err.(*appErr.AppError); ok {
		c.JSON(e.Code, gin.H{"error": gin.H{"code": e.Code, "message": e.Message, "detail": e.Detail, "request_id": c.GetString("X-Request-ID")}})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": "internal server error", "request_id": c.GetString("X-Request-ID")}})
}
