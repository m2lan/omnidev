package adapter

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/omnidev/go-common/crypto"
)

// UserAIConfig represents a user's AI configuration for adapter creation.
type UserAIConfig struct {
	ID       uuid.UUID
	Provider string
	APIKey   string // Encrypted
	BaseURL  string
	Protocol string // "openai" or "anthropic"
	Models   []string
}

// Factory creates adapters from user configurations.
type Factory struct {
	encryptor *crypto.Encryptor
}

// NewFactory creates a new adapter factory.
func NewFactory(encryptor *crypto.Encryptor) *Factory {
	return &Factory{encryptor: encryptor}
}

// CreateAdapter creates an adapter from a user config.
func (f *Factory) CreateAdapter(cfg *UserAIConfig) (Adapter, error) {
	// Decrypt API key
	apiKey, err := f.encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	switch cfg.Protocol {
	case "openai":
		return NewOpenAIAdapterFromConfig(apiKey, cfg.BaseURL, cfg.Models), nil
	case "anthropic":
		return NewAnthropicAdapterFromConfig(apiKey, cfg.BaseURL, cfg.Models), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}

// CreateAdapterFromPlaintext creates an adapter with a plaintext API key (for testing).
func (f *Factory) CreateAdapterFromPlaintext(protocol, apiKey, baseURL string, models []string) (Adapter, error) {
	switch protocol {
	case "openai":
		return NewOpenAIAdapterFromConfig(apiKey, baseURL, models), nil
	case "anthropic":
		return NewAnthropicAdapterFromConfig(apiKey, baseURL, models), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
