package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// ModelConfigInput represents a model configuration.
type ModelConfigInput struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// modelIDs extracts model IDs from ModelConfigInput slice.
func modelIDs(models []ModelConfigInput) []string {
	ids := make([]string, 0, len(models))
	for _, m := range models {
		ids = append(ids, m.ID)
	}
	return ids
}

// CreateConfigInput defines the input for creating an AI config.
type CreateConfigInput struct {
	Provider    string            `json:"provider" binding:"required"`
	DisplayName string            `json:"display_name"`
	APIKey      string            `json:"api_key" binding:"required"`
	BaseURL     string            `json:"base_url" binding:"required"`
	Protocol    string            `json:"protocol" binding:"required"`
	Models      []ModelConfigInput `json:"models"`
}

// UpdateConfigInput defines the input for updating an AI config.
type UpdateConfigInput struct {
	Provider    string            `json:"provider"`
	DisplayName string            `json:"display_name"`
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url"`
	Protocol    string            `json:"protocol"`
	Models      []ModelConfigInput `json:"models"`
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

	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.Provider
	}

	cfg := &adapter.UserAIConfig{
		Provider:    input.Provider,
		DisplayName: displayName,
		APIKey:      apiKey,
		BaseURL:     input.BaseURL,
		Protocol:    input.Protocol,
		Models:      modelIDs(input.Models),
	}

	if err := h.configRepo.Create(c.Request.Context(), cfg, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to create config", "detail": err.Error()},
		})
		return
	}

	// Return response matching frontend type
	type modelConfig struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	models := make([]modelConfig, 0, len(cfg.Models))
	for _, m := range cfg.Models {
		models = append(models, modelConfig{ID: m, DisplayName: m})
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"id":           cfg.ID,
		"user_id":      userID,
		"provider":     cfg.Provider,
		"display_name": cfg.DisplayName,
		"api_key_mask": "***",
		"base_url":     cfg.BaseURL,
		"protocol":     cfg.Protocol,
		"models":       models,
		"is_default":   false,
		"is_active":    true,
		"created_at":   cfg.CreatedAt,
		"updated_at":   cfg.UpdatedAt,
	}})
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

	// Response structure matching frontend UserAIConfig type
	type modelConfig struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	type configResponse struct {
		ID          uuid.UUID    `json:"id"`
		UserID      uuid.UUID    `json:"user_id"`
		Provider    string       `json:"provider"`
		DisplayName string       `json:"display_name"`
		APIKeyMask  string       `json:"api_key_mask"`
		BaseURL     string       `json:"base_url"`
		Protocol    string       `json:"protocol"`
		Models      []modelConfig `json:"models"`
		IsDefault   bool         `json:"is_default"`
		IsActive    bool         `json:"is_active"`
		CreatedAt   string       `json:"created_at"`
		UpdatedAt   string       `json:"updated_at"`
	}

	result := make([]configResponse, 0, len(configs))
	for _, cfg := range configs {
		// Mask API key
		apiKeyMask := "***"
		if cfg.APIKey != "" && h.encryptor != nil {
			decrypted, err := h.encryptor.Decrypt(cfg.APIKey)
			if err == nil && len(decrypted) > 4 {
				apiKeyMask = decrypted[:4] + "***" + decrypted[len(decrypted)-4:]
			}
		} else if cfg.APIKey != "" {
			if len(cfg.APIKey) > 4 {
				apiKeyMask = cfg.APIKey[:4] + "***"
			}
		}

		// Convert models to ModelConfig format
		models := make([]modelConfig, 0, len(cfg.Models))
		for _, m := range cfg.Models {
			models = append(models, modelConfig{ID: m, DisplayName: m})
		}

		result = append(result, configResponse{
			ID:          cfg.ID,
			UserID:      cfg.UserID,
			Provider:    cfg.Provider,
			DisplayName: cfg.DisplayName,
			APIKeyMask:  apiKeyMask,
			BaseURL:     cfg.BaseURL,
			Protocol:    cfg.Protocol,
			Models:      models,
			IsDefault:   cfg.IsDefault,
			IsActive:    cfg.IsActive,
			CreatedAt:   cfg.CreatedAt,
			UpdatedAt:   cfg.UpdatedAt,
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

	// Mask API key
	apiKeyMask := "***"
	if cfg.APIKey != "" && h.encryptor != nil {
		decrypted, err := h.encryptor.Decrypt(cfg.APIKey)
		if err == nil && len(decrypted) > 4 {
			apiKeyMask = decrypted[:4] + "***" + decrypted[len(decrypted)-4:]
		}
	} else if cfg.APIKey != "" {
		if len(cfg.APIKey) > 4 {
			apiKeyMask = cfg.APIKey[:4] + "***"
		}
	}

	// Convert models to ModelConfig format
	type modelConfig struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	models := make([]modelConfig, 0, len(cfg.Models))
	for _, m := range cfg.Models {
		models = append(models, modelConfig{ID: m, DisplayName: m})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":           cfg.ID,
		"user_id":      cfg.UserID,
		"provider":     cfg.Provider,
		"display_name": cfg.DisplayName,
		"api_key_mask": apiKeyMask,
		"base_url":     cfg.BaseURL,
		"protocol":     cfg.Protocol,
		"models":       models,
		"is_default":   cfg.IsDefault,
		"is_active":    cfg.IsActive,
		"created_at":   cfg.CreatedAt,
		"updated_at":   cfg.UpdatedAt,
	}})
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
	if input.DisplayName != "" {
		existing.DisplayName = input.DisplayName
	}
	if input.BaseURL != "" {
		existing.BaseURL = input.BaseURL
	}
	if input.Protocol != "" {
		existing.Protocol = input.Protocol
	}
	if input.Models != nil {
		existing.Models = modelIDs(input.Models)
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

	// Mask API key for response
	apiKeyMask := "***"
	if existing.APIKey != "" && h.encryptor != nil {
		decrypted, err := h.encryptor.Decrypt(existing.APIKey)
		if err == nil && len(decrypted) > 4 {
			apiKeyMask = decrypted[:4] + "***" + decrypted[len(decrypted)-4:]
		}
	} else if existing.APIKey != "" {
		if len(existing.APIKey) > 4 {
			apiKeyMask = existing.APIKey[:4] + "***"
		}
	}

	type modelConfig struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	models := make([]modelConfig, 0, len(existing.Models))
	for _, m := range existing.Models {
		models = append(models, modelConfig{ID: m, DisplayName: m})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":           existing.ID,
		"user_id":      existing.UserID,
		"provider":     existing.Provider,
		"display_name": existing.DisplayName,
		"api_key_mask": apiKeyMask,
		"base_url":     existing.BaseURL,
		"protocol":     existing.Protocol,
		"models":       models,
		"is_default":   existing.IsDefault,
		"is_active":    existing.IsActive,
		"created_at":   existing.CreatedAt,
		"updated_at":   existing.UpdatedAt,
	}})
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

	// Build test URL based on provider and protocol
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	var testURL string
	reqHeaders := map[string]string{
		"Authorization": "Bearer " + apiKey,
	}

	switch {
	case cfg.Protocol == "anthropic":
		testURL = baseURL + "/v1/models"
		reqHeaders["x-api-key"] = apiKey
		reqHeaders["anthropic-version"] = "2023-06-01"
		delete(reqHeaders, "Authorization")
	default: // openai protocol
		testURL = baseURL + "/v1/models"
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", testURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"success": false, "message": "failed to create request", "latency_ms": 0},
		})
		return
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"success": false, "message": fmt.Sprintf("connection failed: %v", err), "latency_ms": latency},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"success": false, "message": fmt.Sprintf("server returned %d", resp.StatusCode), "latency_ms": latency},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"success": true, "message": "connection test passed", "latency_ms": latency},
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
