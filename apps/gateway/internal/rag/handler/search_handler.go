package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/rag/service"
)

// SearchHandler handles search endpoints.
type SearchHandler struct {
	searchSvc *service.SearchService
}

// NewSearchHandler creates a new search handler.
func NewSearchHandler(searchSvc *service.SearchService) *SearchHandler {
	return &SearchHandler{searchSvc: searchSvc}
}

// Search performs a hybrid search on a knowledge base.
// POST /api/v1/knowledge/:id/search
func (h *SearchHandler) Search(c *gin.Context) {
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

	var input service.SearchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	if input.Query == "" {
		badRequest(c, "query is required")
		return
	}

	results, err := h.searchSvc.Search(c.Request.Context(), userID, kbID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
