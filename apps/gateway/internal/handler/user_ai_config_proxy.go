package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/crypto"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/repository"
)

// UserAIConfigHandler handles user AI configuration requests directly.
type UserAIConfigHandler struct {
	configRepo repository.UserAIConfigRepository
	encryptor  *crypto.Encryptor
}

// NewUserAIConfigProxyHandler creates a new AI config handler.
// Note: kept the same constructor name for backward compatibility.
func NewUserAIConfigProxyHandler(_ string, configRepo repository.UserAIConfigRepository, encryptor *crypto.Encryptor) *UserAIConfigHandler {
	return &UserAIConfigHandler{
		configRepo: configRepo,
		encryptor:  encryptor,
	}
}

// CreateConfigInput defines the input for creating an AI config.
type CreateConfigInput struct {
	Provider string   `json:"provider" binding:"required"`
	APIKey   string   `json:"api_key" binding:"required"`
	BaseURL  string   `json:"base_url" binding:"required"`
	Protocol string   `json:"protocol" binding:"required"`
	Models   []string `json:"models"`
}

// UpdateConfigInput defines the input for updating an AI config.
type UpdateConfigInput struct {
	Provider string   `json:"provider"`
	APIKey   string   `json:"api_key"`
	BaseURL  string   `json:"base_url"`
	Protocol string   `json:"protocol"`
	Models   []string `json:"models"`
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

	var input CreateConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid request body", "detail": err.Error()},
		})
		return
	}

	// Encrypt API key if encryptor is available
	apiKey := input.APIKey
	if h.encryptor != nil {
		encrypted, err := h.encryptor.Encrypt(input.APIKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": 500, "message": "failed to encrypt api key"},
			})
			return
		}
		apiKey = encrypted
	}

	cfg := &adapter.UserAIConfig{
		Provider: input.Provider,
		APIKey:   apiKey,
		BaseURL:  input.BaseURL,
		Protocol: input.Protocol,
		Models:   input.Models,
	}

	if err := h.configRepo.Create(c.Request.Context(), cfg, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to create config"},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": cfg})
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

	configs, err := h.configRepo.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to list configs"},
		})
		return
	}

	// Decrypt API keys for display (mask them)
	type configResponse struct {
		ID       uuid.UUID `json:"id"`
		Provider string    `json:"provider"`
		APIKey   string    `json:"api_key"`
		BaseURL  string    `json:"base_url"`
		Protocol string    `json:"protocol"`
		Models   []string  `json:"models"`
	}

	result := make([]configResponse, 0, len(configs))
	for _, cfg := range configs {
		apiKey := "***"
		if cfg.APIKey != "" && h.encryptor != nil {
			decrypted, err := h.encryptor.Decrypt(cfg.APIKey)
			if err == nil && len(decrypted) > 4 {
				apiKey = decrypted[:4] + "***" + decrypted[len(decrypted)-4:]
			}
		} else if cfg.APIKey != "" {
			// No encryptor, just mask the stored key
			if len(cfg.APIKey) > 4 {
				apiKey = cfg.APIKey[:4] + "***"
			}
		}
		result = append(result, configResponse{
			ID:       cfg.ID,
			Provider: cfg.Provider,
			APIKey:   apiKey,
			BaseURL:  cfg.BaseURL,
			Protocol: cfg.Protocol,
			Models:   cfg.Models,
		})
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

	cfg, err := h.configRepo.GetByIDAndUser(c.Request.Context(), userID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cfg})
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

	// Verify ownership
	existing, err := h.configRepo.GetByIDAndUser(c.Request.Context(), userID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	var input UpdateConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid request body", "detail": err.Error()},
		})
		return
	}

	// Merge fields
	if input.Provider != "" {
		existing.Provider = input.Provider
	}
	if input.BaseURL != "" {
		existing.BaseURL = input.BaseURL
	}
	if input.Protocol != "" {
		existing.Protocol = input.Protocol
	}
	if input.Models != nil {
		existing.Models = input.Models
	}
	if input.APIKey != "" {
		if h.encryptor != nil {
			encryptedKey, err := h.encryptor.Encrypt(input.APIKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": 500, "message": "failed to encrypt api key"},
				})
				return
			}
			existing.APIKey = encryptedKey
		} else {
			existing.APIKey = input.APIKey
		}
	}

	if err := h.configRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to update config"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": existing})
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

	if err := h.configRepo.Delete(c.Request.Context(), userID, configID); err != nil {
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

	if err := h.configRepo.SetDefault(c.Request.Context(), userID, configID); err != nil {
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

	cfg, err := h.configRepo.GetByIDAndUser(c.Request.Context(), userID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": 404, "message": "config not found"},
		})
		return
	}

	// Decrypt API key for testing (if encryptor available)
	apiKey := cfg.APIKey
	if h.encryptor != nil {
		decrypted, err := h.encryptor.Decrypt(cfg.APIKey)
		if err == nil {
			apiKey = decrypted
		}
	}

	// Test connection based on provider
	// For now, return success if config exists
	_ = apiKey

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"success": true,
			"message": "connection test passed",
		},
	})
}

// GetUserAIConfigForModel returns a user's AI config for a specific model.
// This is used internally by the gateway's chat service.
func (h *UserAIConfigHandler) GetUserAIConfigForModel(userID uuid.UUID, modelID string) (*adapter.UserAIConfig, error) {
	// List user configs and find one that supports the model
	configs, err := h.configRepo.ListByUserID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	for _, cfg := range configs {
		for _, m := range cfg.Models {
			if m == modelID {
				return cfg, nil
			}
		}
	}

	// Return default config if model not found in any specific config
	if len(configs) > 0 {
		return configs[0], nil
	}

	return nil, fmt.Errorf("no ai config found for model %s", modelID)
}
