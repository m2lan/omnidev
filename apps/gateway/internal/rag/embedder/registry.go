// Package embedder provides text embedding capabilities with a pluggable provider registry.
package embedder

import (
	"fmt"
	"net/http"
	"time"
)

// EmbedderConfig holds configuration for creating an embedder.
// It is provider-agnostic; each provider picks the fields it needs.
type EmbedderConfig struct {
	// Provider is the embedding provider name (e.g. "openai", "gemini", "ollama").
	Provider string
	// Model is the specific model identifier (e.g. "text-embedding-3-small").
	Model string
	// APIKey for authentication. Providers fall back to their own config if empty.
	APIKey string
	// BaseURL for self-hosted or compatible endpoints (Ollama, vLLM, etc.).
	BaseURL string
	// Dimensions override. 0 means use the provider default.
	Dimensions int
	// HTTPClient override. nil means use a default 30s client.
	HTTPClient *http.Client
}

// Factory creates an Embedder from a config.
type Factory func(cfg EmbedderConfig) (Embedder, error)

var registry = map[string]Factory{}

// Register adds an embedding provider factory under the given name.
// Call this from init() in each provider file.
func Register(provider string, f Factory) {
	registry[provider] = f
}

// Providers returns the list of registered provider names.
func Providers() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}

// New creates an Embedder using the registered factory for cfg.Provider.
// Returns an error if the provider is unknown or the factory fails.
func New(cfg EmbedderConfig) (Embedder, error) {
	f, ok := registry[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown embedding provider %q, registered: %v", cfg.Provider, Providers())
	}
	return f(cfg)
}

// defaultHTTPClient returns a standard HTTP client with 30s timeout.
func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func init() {
	// --- OpenAI (and OpenAI-compatible: vLLM, text-embedding-inference, etc.) ---
	Register("openai", func(cfg EmbedderConfig) (Embedder, error) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai embedding provider requires api_key")
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model := cfg.Model
		if model == "" {
			model = "text-embedding-3-small"
		}
		dims := cfg.Dimensions
		if dims == 0 {
			dims = openaiDefaultDimensions(model)
		}
		client := cfg.HTTPClient
		if client == nil {
			client = defaultHTTPClient()
		}
		return &OpenAIEmbedder{
			apiKey:     cfg.APIKey,
			baseURL:    baseURL,
			model:      model,
			dimensions: dims,
			client:     client,
		}, nil
	})

	// --- Gemini ---
	Register("gemini", func(cfg EmbedderConfig) (Embedder, error) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("gemini embedding provider requires api_key")
		}
		model := cfg.Model
		if model == "" {
			model = "gemini-embedding-2"
		}
		dims := cfg.Dimensions
		if dims == 0 {
			dims = 768
		}
		client := cfg.HTTPClient
		if client == nil {
			client = defaultHTTPClient()
		}
		return &GeminiEmbedder{
			apiKey:     cfg.APIKey,
			model:      model,
			dimensions: dims,
			client:     client,
		}, nil
	})

	// --- Ollama (OpenAI-compatible API) ---
	Register("ollama", func(cfg EmbedderConfig) (Embedder, error) {
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		model := cfg.Model
		if model == "" {
			model = "nomic-embed-text"
		}
		dims := cfg.Dimensions
		if dims == 0 {
			dims = 768
		}
		client := cfg.HTTPClient
		if client == nil {
			client = defaultHTTPClient()
		}
		// Ollama uses OpenAI-compatible /embeddings endpoint, no API key needed.
		return &OpenAIEmbedder{
			apiKey:     cfg.APIKey, // typically empty for Ollama
			baseURL:    baseURL,
			model:      model,
			dimensions: dims,
			client:     client,
		}, nil
	})
}

// openaiDefaultDimensions returns the default dimension count for known OpenAI models.
func openaiDefaultDimensions(model string) int {
	switch model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536
	}
}
