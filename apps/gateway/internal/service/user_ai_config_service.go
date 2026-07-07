package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/omnidev/go-common/crypto"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/repository"
)

// ModelConfigInput represents a model configuration.
type ModelConfigInput struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
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

// ConfigResponse represents the API response for a user AI config.
type ConfigResponse struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Provider    string         `json:"provider"`
	DisplayName string         `json:"display_name"`
	APIKeyMask  string         `json:"api_key_mask"`
	BaseURL     string         `json:"base_url"`
	Protocol    string         `json:"protocol"`
	Models      []ModelConfig  `json:"models"`
	IsDefault   bool           `json:"is_default"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// ModelConfig represents a model in the response.
type ModelConfig struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// TestConnectionResult represents the result of a connection test.
type TestConnectionResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
}

// UserAIConfigService handles user AI configuration business logic.
type UserAIConfigService struct {
	configRepo repository.UserAIConfigRepository
	encryptor  *crypto.Encryptor
}

// NewUserAIConfigService creates a new user AI config service.
func NewUserAIConfigService(
	configRepo repository.UserAIConfigRepository,
	encryptor *crypto.Encryptor,
) *UserAIConfigService {
	return &UserAIConfigService{
		configRepo: configRepo,
		encryptor:  encryptor,
	}
}

// Create creates a new user AI config.
func (s *UserAIConfigService) Create(ctx context.Context, userID uuid.UUID, input *CreateConfigInput) (*ConfigResponse, error) {
	// Encrypt API key if encryptor is available
	apiKey := input.APIKey
	if s.encryptor != nil {
		encrypted, err := s.encryptor.Encrypt(input.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt api key: %w", err)
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
		Models:      modelConfigIDs(input.Models),
	}

	if err := s.configRepo.Create(ctx, cfg, userID); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return &ConfigResponse{
		ID:          cfg.ID,
		UserID:      userID,
		Provider:    cfg.Provider,
		DisplayName: cfg.DisplayName,
		APIKeyMask:  "***",
		BaseURL:     cfg.BaseURL,
		Protocol:    cfg.Protocol,
		Models:      toModelConfigs(cfg.Models),
		IsDefault:   false,
		IsActive:    true,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt,
	}, nil
}

// List returns all AI configs for a user.
func (s *UserAIConfigService) List(ctx context.Context, userID uuid.UUID) ([]ConfigResponse, error) {
	configs, err := s.configRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	result := make([]ConfigResponse, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, ConfigResponse{
			ID:          cfg.ID,
			UserID:      cfg.UserID,
			Provider:    cfg.Provider,
			DisplayName: cfg.DisplayName,
			APIKeyMask:  s.maskAPIKey(cfg.APIKey),
			BaseURL:     cfg.BaseURL,
			Protocol:    cfg.Protocol,
			Models:      toModelConfigs(cfg.Models),
			IsDefault:   cfg.IsDefault,
			IsActive:    cfg.IsActive,
			CreatedAt:   cfg.CreatedAt,
			UpdatedAt:   cfg.UpdatedAt,
		})
	}

	return result, nil
}

// Get returns a specific AI config by ID with ownership verification.
func (s *UserAIConfigService) Get(ctx context.Context, userID, configID uuid.UUID) (*ConfigResponse, error) {
	cfg, err := s.configRepo.GetByIDAndUser(ctx, userID, configID)
	if err != nil {
		return nil, fmt.Errorf("config not found")
	}

	return &ConfigResponse{
		ID:          cfg.ID,
		UserID:      cfg.UserID,
		Provider:    cfg.Provider,
		DisplayName: cfg.DisplayName,
		APIKeyMask:  s.maskAPIKey(cfg.APIKey),
		BaseURL:     cfg.BaseURL,
		Protocol:    cfg.Protocol,
		Models:      toModelConfigs(cfg.Models),
		IsDefault:   cfg.IsDefault,
		IsActive:    cfg.IsActive,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt,
	}, nil
}

// Update updates an existing AI config with ownership verification.
func (s *UserAIConfigService) Update(ctx context.Context, userID, configID uuid.UUID, input *UpdateConfigInput) (*ConfigResponse, error) {
	existing, err := s.configRepo.GetByIDAndUser(ctx, userID, configID)
	if err != nil {
		return nil, fmt.Errorf("config not found")
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
		existing.Models = modelConfigIDs(input.Models)
	}
	if input.APIKey != "" {
		if s.encryptor != nil {
			encryptedKey, err := s.encryptor.Encrypt(input.APIKey)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt api key: %w", err)
			}
			existing.APIKey = encryptedKey
		} else {
			existing.APIKey = input.APIKey
		}
	}

	if err := s.configRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return &ConfigResponse{
		ID:          existing.ID,
		UserID:      existing.UserID,
		Provider:    existing.Provider,
		DisplayName: existing.DisplayName,
		APIKeyMask:  s.maskAPIKey(existing.APIKey),
		BaseURL:     existing.BaseURL,
		Protocol:    existing.Protocol,
		Models:      toModelConfigs(existing.Models),
		IsDefault:   existing.IsDefault,
		IsActive:    existing.IsActive,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	}, nil
}

// Delete soft-deletes an AI config with ownership verification.
func (s *UserAIConfigService) Delete(ctx context.Context, userID, configID uuid.UUID) error {
	return s.configRepo.Delete(ctx, userID, configID)
}

// SetDefault sets an AI config as the default for a user.
func (s *UserAIConfigService) SetDefault(ctx context.Context, userID, configID uuid.UUID) error {
	return s.configRepo.SetDefault(ctx, userID, configID)
}

// TestConnection tests connectivity to an AI provider.
func (s *UserAIConfigService) TestConnection(ctx context.Context, userID, configID uuid.UUID) (*TestConnectionResult, error) {
	cfg, err := s.configRepo.GetByIDAndUser(ctx, userID, configID)
	if err != nil {
		return nil, fmt.Errorf("config not found")
	}

	// Decrypt API key for testing
	apiKey := cfg.APIKey
	if s.encryptor != nil {
		decrypted, err := s.encryptor.Decrypt(cfg.APIKey)
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
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return &TestConnectionResult{Success: false, Message: "failed to create request"}, nil
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestConnectionResult{Success: false, Message: fmt.Sprintf("connection failed: %v", err), LatencyMs: latency}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &TestConnectionResult{Success: false, Message: fmt.Sprintf("server returned %d", resp.StatusCode), LatencyMs: latency}, nil
	}

	return &TestConnectionResult{Success: true, Message: "connection test passed", LatencyMs: latency}, nil
}

// GetUserAIConfigForModel returns a user's AI config for a specific model.
// Used internally by the gateway's chat service.
func (s *UserAIConfigService) GetUserAIConfigForModel(userID uuid.UUID, modelID string) (*adapter.UserAIConfig, error) {
	configs, err := s.configRepo.ListByUserID(context.Background(), userID)
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

// maskAPIKey masks an API key for safe display.
func (s *UserAIConfigService) maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "***"
	}

	// Try to decrypt first for proper masking
	if s.encryptor != nil {
		decrypted, err := s.encryptor.Decrypt(apiKey)
		if err == nil && len(decrypted) > 4 {
			return decrypted[:4] + "***" + decrypted[len(decrypted)-4:]
		}
	}

	if len(apiKey) > 4 {
		return apiKey[:4] + "***"
	}
	return "***"
}

// modelConfigIDs extracts model IDs from ModelConfigInput slice.
func modelConfigIDs(models []ModelConfigInput) []string {
	ids := make([]string, 0, len(models))
	for _, m := range models {
		ids = append(ids, m.ID)
	}
	return ids
}

// toModelConfigs converts model ID strings to ModelConfig slice.
func toModelConfigs(models []string) []ModelConfig {
	configs := make([]ModelConfig, 0, len(models))
	for _, m := range models {
		configs = append(configs, ModelConfig{ID: m, DisplayName: m})
	}
	return configs
}
