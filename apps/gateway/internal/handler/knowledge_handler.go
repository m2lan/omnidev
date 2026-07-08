// Package handler provides HTTP handlers for the API Gateway.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	knowledge "github.com/omnidev/knowledge-engine"
)

// KnowledgeHandler wraps knowledge-engine services with Gin-compatible handlers.
type KnowledgeHandler struct {
	kbSvc     *knowledge.KnowledgeBaseService
	searchSvc *knowledge.SearchService
}

// NewKnowledgeHandler creates a new knowledge handler.
func NewKnowledgeHandler(kbSvc *knowledge.KnowledgeBaseService, searchSvc *knowledge.SearchService) *KnowledgeHandler {
	return &KnowledgeHandler{kbSvc: kbSvc, searchSvc: searchSvc}
}

// ListKnowledgeBases returns a paginated list of knowledge bases.
func (h *KnowledgeHandler) ListKnowledgeBases(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	kbs, total, err := h.kbSvc.ListKnowledgeBases(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": kbs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// CreateKnowledgeBase creates a new knowledge base.
func (h *KnowledgeHandler) CreateKnowledgeBase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var input knowledge.CreateKBInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	kb, err := h.kbSvc.CreateKnowledgeBase(c.Request.Context(), userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": kb})
}

// GetKnowledgeBase returns a knowledge base by ID.
func (h *KnowledgeHandler) GetKnowledgeBase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	kb, err := h.kbSvc.GetKnowledgeBase(c.Request.Context(), userID, kbID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": 404, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": kb})
}

// UpdateKnowledgeBase updates a knowledge base.
func (h *KnowledgeHandler) UpdateKnowledgeBase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	var input knowledge.CreateKBInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	kb, err := h.kbSvc.UpdateKnowledgeBase(c.Request.Context(), userID, kbID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": kb})
}

// DeleteKnowledgeBase deletes a knowledge base.
func (h *KnowledgeHandler) DeleteKnowledgeBase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	if err := h.kbSvc.DeleteKnowledgeBase(c.Request.Context(), userID, kbID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "knowledge base deleted"}})
}

// ListDocuments returns documents in a knowledge base.
func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	docs, total, err := h.kbSvc.ListDocuments(c.Request.Context(), userID, kbID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": docs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// UploadDocument uploads a document to a knowledge base.
func (h *KnowledgeHandler) UploadDocument(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "file is required"}})
		return
	}
	defer file.Close()

	doc, err := h.kbSvc.UploadDocument(c.Request.Context(), userID, kbID, header.Filename, header.Size, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": doc})
}

// DeleteDocument deletes a document.
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid document ID"}})
		return
	}

	if err := h.kbSvc.DeleteDocument(c.Request.Context(), userID, kbID, docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "document deleted"}})
}

// Search performs a hybrid search on a knowledge base.
func (h *KnowledgeHandler) Search(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "invalid knowledge base ID"}})
		return
	}

	var input knowledge.SearchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	results, err := h.searchSvc.Search(c.Request.Context(), userID, kbID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

// getUserID extracts the user ID from the Gin context.
func getUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
		return uuid.Nil, false
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
		return uuid.Nil, false
	}
	return userID, true
}
