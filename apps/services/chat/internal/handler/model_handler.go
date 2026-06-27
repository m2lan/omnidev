package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omnidev/services/chat/internal/repository"
)

// ModelHandler handles model endpoints.
type ModelHandler struct {
	modelRepo repository.ModelRepository
}

// NewModelHandler creates a new model handler.
func NewModelHandler(modelRepo repository.ModelRepository) *ModelHandler {
	return &ModelHandler{modelRepo: modelRepo}
}

// ListModels returns available AI models.
// GET /api/v1/models
func (h *ModelHandler) ListModels(c *gin.Context) {
	models, err := h.modelRepo.List(c.Request.Context(), true)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}
