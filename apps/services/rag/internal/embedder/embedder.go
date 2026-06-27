// Package embedder provides text embedding capabilities.
package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

// Embedder defines the interface for text embedding.
type Embedder interface {
	// Embed generates embeddings for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Model returns the embedding model name.
	Model() string

	// Dimensions returns the embedding dimensions.
	Dimensions() int
}

// OpenAIEmbedder implements the Embedder interface using OpenAI's embedding API.
type OpenAIEmbedder struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	client     *http.Client
}

// NewOpenAIEmbedder creates a new OpenAI embedder.
func NewOpenAIEmbedder(cfg config.AIProviderConfig) *OpenAIEmbedder {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIEmbedder{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		model:      "text-embedding-3-small",
		dimensions: 1536,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *OpenAIEmbedder) Model() string    { return e.model }
func (e *OpenAIEmbedder) Dimensions() int  { return e.dimensions }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return vectors[0], nil
}

func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// OpenAI has a batch limit of 2048
	const maxBatch = 2048
	allVectors := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += maxBatch {
		end := i + maxBatch
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		vectors, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		allVectors = append(allVectors, vectors...)
	}

	return allVectors, nil
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body := map[string]interface{}{
		"model": e.model,
		"input": texts,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result openaiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	vectors := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		vec := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			vec[j] = float32(v)
		}
		vectors[i] = vec
	}

	logger.Log.Debug("Embeddings generated",
		zap.Int("batch_size", len(texts)),
		zap.Int("dimensions", len(vectors[0])),
	)

	return vectors, nil
}

type openaiEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
