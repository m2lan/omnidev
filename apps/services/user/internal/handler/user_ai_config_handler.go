package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/user/internal/service"
)

// UserAIConfigHandler handles user AI configuration endpoints.
type UserAIConfigHandler struct {
	configSvc *service.UserAIConfigService
}

// NewUserAIConfigHandler creates a new user AI config handler.
func NewUserAIConfigHandler(configSvc *service.UserAIConfigService) *UserAIConfigHandler {
	return &UserAIConfigHandler{configSvc: configSvc}
}

// Create creates a new AI config.
// POST /api/v1/user/ai-configs
func (h *UserAIConfigHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input service.CreateConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid request body", "detail": err.Error()},
		})
		return
	}

	config, err := h.configSvc.Create(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": config,
	})
}

// List returns all AI configs for the authenticated user.
// GET /api/v1/user/ai-configs
func (h *UserAIConfigHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configs, err := h.configSvc.List(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": configs,
	})
}

// Get returns a specific AI config.
// GET /api/v1/user/ai-configs/:id
func (h *UserAIConfigHandler) Get(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid config id"},
		})
		return
	}

	config, err := h.configSvc.Get(c.Request.Context(), userID, configID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": config,
	})
}

// Update updates an AI config.
// PUT /api/v1/user/ai-configs/:id
func (h *UserAIConfigHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid config id"},
		})
		return
	}

	var input service.UpdateConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid request body", "detail": err.Error()},
		})
		return
	}

	config, err := h.configSvc.Update(c.Request.Context(), userID, configID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": config,
	})
}

// Delete deletes an AI config.
// DELETE /api/v1/user/ai-configs/:id
func (h *UserAIConfigHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid config id"},
		})
		return
	}

	if err := h.configSvc.Delete(c.Request.Context(), userID, configID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "config deleted"},
	})
}

// SetDefault sets an AI config as default.
// PUT /api/v1/user/ai-configs/:id/default
func (h *UserAIConfigHandler) SetDefault(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid config id"},
		})
		return
	}

	if err := h.configSvc.SetDefault(c.Request.Context(), userID, configID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "default config set"},
	})
}

// TestConnection tests the connection to an AI provider.
// POST /api/v1/user/ai-configs/:id/test
func (h *UserAIConfigHandler) TestConnection(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	configID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid config id"},
		})
		return
	}

	result, err := h.configSvc.TestConnection(c.Request.Context(), userID, configID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}
