package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/crypto"
	"github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/repository"
)

// UserAIConfigService handles user AI configuration operations.
type UserAIConfigService struct {
	configRepo repository.UserAIConfigRepository
	encryptor  *crypto.Encryptor
	cache      *cache.Redis
	httpClient *http.Client
}

// NewUserAIConfigService creates a new user AI config service.
func NewUserAIConfigService(
	configRepo repository.UserAIConfigRepository,
	encryptor *crypto.Encryptor,
	cache *cache.Redis,
) *UserAIConfigService {
	return &UserAIConfigService{
		configRepo: configRepo,
		encryptor:  encryptor,
		cache:      cache,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateConfigInput defines the input for creating an AI config.
type CreateConfigInput struct {
	Provider    string              `json:"provider" validate:"required"`
	DisplayName string              `json:"display_name" validate:"required"`
	APIKey      string              `json:"api_key" validate:"required"`
	BaseURL     string              `json:"base_url" validate:"required"`
	Protocol    domain.AIProtocol   `json:"protocol" validate:"required,oneof=openai anthropic"`
	Models      []domain.ModelConfig `json:"models"`
	IsDefault   bool                `json:"is_default"`
}

// UpdateConfigInput defines the input for updating an AI config.
type UpdateConfigInput struct {
	DisplayName *string              `json:"display_name,omitempty"`
	APIKey      *string              `json:"api_key,omitempty"`
	BaseURL     *string              `json:"base_url,omitempty"`
	Protocol    *domain.AIProtocol   `json:"protocol,omitempty"`
	Models      []domain.ModelConfig `json:"models,omitempty"`
	IsDefault   *bool                `json:"is_default,omitempty"`
	IsActive    *bool                `json:"is_active,omitempty"`
}

// ConfigResponse represents the API response for a config.
type ConfigResponse struct {
	ID          uuid.UUID           `json:"id"`
	UserID      uuid.UUID           `json:"user_id"`
	Provider    string              `json:"provider"`
	DisplayName string              `json:"display_name"`
	APIKeyMask  string              `json:"api_key_mask"`
	BaseURL     string              `json:"base_url"`
	Protocol    domain.AIProtocol   `json:"protocol"`
	Models      []domain.ModelConfig `json:"models"`
	IsDefault   bool                `json:"is_default"`
	IsActive    bool                `json:"is_active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// TestResult represents the result of a connection test.
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Latency int    `json:"latency_ms"`
}

// Create creates a new AI config for a user.
func (s *UserAIConfigService) Create(ctx context.Context, userID uuid.UUID, input *CreateConfigInput) (*ConfigResponse, error) {
	// Encrypt API key
	encryptedKey, err := s.encryptor.Encrypt(input.APIKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt API key")
	}

	config := &domain.UserAIConfig{
		ID:          uuid.New(),
		UserID:      userID,
		Provider:    input.Provider,
		DisplayName: input.DisplayName,
		APIKey:      encryptedKey,
		BaseURL:     input.BaseURL,
		Protocol:    input.Protocol,
		Models:      input.Models,
		IsDefault:   input.IsDefault,
		IsActive:    true,
	}

	// If setting as default, clear other defaults first
	if input.IsDefault {
		if err := s.configRepo.ClearDefault(ctx, userID); err != nil {
			return nil, errors.Wrap(err, "failed to clear default configs")
		}
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return nil, errors.Wrap(err, "failed to create config")
	}

	// Invalidate cache
	s.invalidateCache(ctx, userID)

	logger.Log.Info("AI config created",
		zap.String("user_id", userID.String()),
		zap.String("provider", input.Provider),
		zap.String("config_id", config.ID.String()),
	)

	return s.toResponse(config), nil
}

// Get returns a specific AI config.
func (s *UserAIConfigService) Get(ctx context.Context, userID, configID uuid.UUID) (*ConfigResponse, error) {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, errors.NotFound("ai config")
	}
	if config.UserID != userID {
		return nil, errors.ErrForbidden
	}
	return s.toResponse(config), nil
}

// List returns all AI configs for a user.
func (s *UserAIConfigService) List(ctx context.Context, userID uuid.UUID) ([]*ConfigResponse, error) {
	configs, err := s.configRepo.ListByUserID(ctx, userID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list configs")
	}

	responses := make([]*ConfigResponse, 0, len(configs))
	for _, config := range configs {
		responses = append(responses, s.toResponse(config))
	}
	return responses, nil
}

// Update updates an AI config.
func (s *UserAIConfigService) Update(ctx context.Context, userID, configID uuid.UUID, input *UpdateConfigInput) (*ConfigResponse, error) {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, errors.NotFound("ai config")
	}
	if config.UserID != userID {
		return nil, errors.ErrForbidden
	}

	update := &domain.UserAIConfigUpdate{
		DisplayName: input.DisplayName,
		BaseURL:     input.BaseURL,
		Protocol:    input.Protocol,
		Models:      input.Models,
		IsDefault:   input.IsDefault,
		IsActive:    input.IsActive,
	}

	// Encrypt API key if provided
	if input.APIKey != nil {
		encryptedKey, err := s.encryptor.Encrypt(*input.APIKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encrypt API key")
		}
		update.APIKey = &encryptedKey
	}

	// If setting as default, clear other defaults first
	if input.IsDefault != nil && *input.IsDefault {
		if err := s.configRepo.ClearDefault(ctx, userID); err != nil {
			return nil, errors.Wrap(err, "failed to clear default configs")
		}
	}

	if err := s.configRepo.Update(ctx, configID, update); err != nil {
		return nil, errors.Wrap(err, "failed to update config")
	}

	// Invalidate cache
	s.invalidateCache(ctx, userID)

	logger.Log.Info("AI config updated",
		zap.String("user_id", userID.String()),
		zap.String("config_id", configID.String()),
	)

	updated, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updated config")
	}
	return s.toResponse(updated), nil
}

// Delete soft-deletes an AI config.
func (s *UserAIConfigService) Delete(ctx context.Context, userID, configID uuid.UUID) error {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return errors.NotFound("ai config")
	}
	if config.UserID != userID {
		return errors.ErrForbidden
	}

	if err := s.configRepo.Delete(ctx, configID); err != nil {
		return errors.Wrap(err, "failed to delete config")
	}

	// Invalidate cache
	s.invalidateCache(ctx, userID)

	logger.Log.Info("AI config deleted",
		zap.String("user_id", userID.String()),
		zap.String("config_id", configID.String()),
	)

	return nil
}

// SetDefault sets a config as the default for a user.
func (s *UserAIConfigService) SetDefault(ctx context.Context, userID, configID uuid.UUID) error {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return errors.NotFound("ai config")
	}
	if config.UserID != userID {
		return errors.ErrForbidden
	}

	// Clear other defaults
	if err := s.configRepo.ClearDefault(ctx, userID); err != nil {
		return errors.Wrap(err, "failed to clear default configs")
	}

	// Set this one as default
	if err := s.configRepo.Update(ctx, configID, &domain.UserAIConfigUpdate{IsDefault: boolPtr(true)}); err != nil {
		return errors.Wrap(err, "failed to set default config")
	}

	// Invalidate cache
	s.invalidateCache(ctx, userID)

	return nil
}

// TestConnection tests the connection to an AI provider.
func (s *UserAIConfigService) TestConnection(ctx context.Context, userID, configID uuid.UUID) (*TestResult, error) {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, errors.NotFound("ai config")
	}
	if config.UserID != userID {
		return nil, errors.ErrForbidden
	}

	// Decrypt API key
	apiKey, err := s.encryptor.Decrypt(config.APIKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt API key")
	}

	start := time.Now()
	success, message := s.doTestConnection(ctx, config.Protocol, config.BaseURL, apiKey)
	latency := int(time.Since(start).Milliseconds())

	return &TestResult{
		Success: success,
		Message: message,
		Latency: latency,
	}, nil
}

// GetDefaultConfig returns the default AI config for a user.
func (s *UserAIConfigService) GetDefaultConfig(ctx context.Context, userID uuid.UUID) (*domain.UserAIConfig, error) {
	config, err := s.configRepo.GetDefault(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("default ai config")
	}
	return config, nil
}

// GetConfigForModel returns the appropriate adapter config for a given model.
// It checks user configs first, then returns nil for global fallback.
func (s *UserAIConfigService) GetConfigForModel(ctx context.Context, userID uuid.UUID, modelID string) (*domain.UserAIConfig, error) {
	configs, err := s.configRepo.ListByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		if !config.IsActive {
			continue
		}
		for _, m := range config.Models {
			if m.ID == modelID {
				return config, nil
			}
		}
	}

	return nil, fmt.Errorf("no user config found for model: %s", modelID)
}

// doTestConnection performs the actual connection test.
func (s *UserAIConfigService) doTestConnection(ctx context.Context, protocol domain.AIProtocol, baseURL, apiKey string) (bool, string) {
	switch protocol {
	case domain.AIProtocolOpenAI:
		return s.testOpenAIConnection(ctx, baseURL, apiKey)
	case domain.AIProtocolAnthropic:
		return s.testAnthropicConnection(ctx, baseURL, apiKey)
	default:
		return false, "unsupported protocol: " + string(protocol)
	}
}

// testOpenAIConnection tests connection to an OpenAI-compatible API.
func (s *UserAIConfigService) testOpenAIConnection(ctx context.Context, baseURL, apiKey string) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return false, "failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, "connection failed: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "connection successful"
	}

	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error.Message != "" {
		return false, errResp.Error.Message
	}
	return false, fmt.Sprintf("API returned status %d", resp.StatusCode)
}

// testAnthropicConnection tests connection to the Anthropic API.
func (s *UserAIConfigService) testAnthropicConnection(ctx context.Context, baseURL, apiKey string) (bool, string) {
	// Anthropic doesn't have a /models endpoint, so we send a minimal messages request
	body := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}
	data, _ := json.Marshal(body)

	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return false, "failed to create request: " + err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, "connection failed: " + err.Error()
	}
	defer resp.Body.Close()

	// Anthropic returns 200 on success, or 401 on auth failure
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		// 400 means auth passed but request was invalid — still a valid connection
		return true, "connection successful"
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return false, "invalid API key"
	}

	return false, fmt.Sprintf("API returned status %d", resp.StatusCode)
}

// toResponse converts a domain config to an API response.
func (s *UserAIConfigService) toResponse(config *domain.UserAIConfig) *ConfigResponse {
	return &ConfigResponse{
		ID:          config.ID,
		UserID:      config.UserID,
		Provider:    config.Provider,
		DisplayName: config.DisplayName,
		APIKeyMask:  maskAPIKey(config.APIKey),
		BaseURL:     config.BaseURL,
		Protocol:    config.Protocol,
		Models:      config.Models,
		IsDefault:   config.IsDefault,
		IsActive:    config.IsActive,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
}

// maskAPIKey masks the API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// invalidateCache invalidates the user's AI config cache.
func (s *UserAIConfigService) invalidateCache(ctx context.Context, userID uuid.UUID) {
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:ai-configs:%s", userID.String()))
}

func boolPtr(b bool) *bool {
	return &b
}
