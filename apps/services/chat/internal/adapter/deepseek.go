package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omnidev/go-common/config"
)

// DeepSeekAdapter implements the Adapter interface for DeepSeek.
// DeepSeek uses OpenAI-compatible API.
type DeepSeekAdapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewDeepSeekAdapter creates a new DeepSeek adapter.
func NewDeepSeekAdapter(cfg config.AIProviderConfig) *DeepSeekAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	return &DeepSeekAdapter{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (a *DeepSeekAdapter) Provider() string {
	return "deepseek"
}

func (a *DeepSeekAdapter) Models() []string {
	return []string{
		"deepseek-chat",
		"deepseek-coder",
		"deepseek-reasoner",
	}
}

func (a *DeepSeekAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result openaiResponse // DeepSeek uses OpenAI-compatible format
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:      result.ID,
		Content: result.Choices[0].Message.Content,
		Model:   result.Model,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		FinishReason: result.Choices[0].FinishReason,
	}, nil
}

func (a *DeepSeekAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	// DeepSeek uses the same SSE format as OpenAI
	// Delegate to OpenAI-compatible streaming
	adapter := &OpenAIAdapter{
		apiKey:  a.apiKey,
		baseURL: a.baseURL,
		client:  a.client,
	}
	return adapter.ChatStream(ctx, req)
}

func (a *DeepSeekAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}
