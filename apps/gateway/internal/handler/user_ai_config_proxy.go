package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/service"
)

// UserAIConfigHandler handles user AI configuration requests.
type UserAIConfigHandler struct {
	svc *service.UserAIConfigService
}

// NewUserAIConfigProxyHandler creates a new AI config handler.
func NewUserAIConfigProxyHandler(svc *service.UserAIConfigService) *UserAIConfigHandler {
	return &UserAIConfigHandler{svc: svc}
}

// Create handles POST /api/v1/user/ai-configs
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

	result, err := h.svc.Create(c.Request.Context(), userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

// List handles GET /api/v1/user/ai-configs
func (h *UserAIConfigHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	result, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to list configs"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Get handles GET /api/v1/user/ai-configs/:id
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

	result, err := h.svc.Get(c.Request.Context(), userID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Update handles PUT /api/v1/user/ai-configs/:id
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

	result, err := h.svc.Update(c.Request.Context(), userID, configID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Delete handles DELETE /api/v1/user/ai-configs/:id
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

	if err := h.svc.Delete(c.Request.Context(), userID, configID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "config deleted"}})
}

// SetDefault handles PUT /api/v1/user/ai-configs/:id/default
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

	if err := h.svc.SetDefault(c.Request.Context(), userID, configID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "default config set"}})
}

// TestConnection handles POST /api/v1/user/ai-configs/:id/test
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

	result, err := h.svc.TestConnection(c.Request.Context(), userID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
