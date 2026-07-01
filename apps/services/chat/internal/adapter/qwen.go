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

// QwenAdapter implements the Adapter interface for Alibaba Qwen.
type QwenAdapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewQwenAdapter creates a new Qwen adapter.
func NewQwenAdapter(cfg config.AIProviderConfig) *QwenAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	return &QwenAdapter{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for streaming
		},
	}
}

func (a *QwenAdapter) Provider() string {
	return "qwen"
}

func (a *QwenAdapter) Models() []string {
	return []string{
		"qwen-turbo",
		"qwen-plus",
		"qwen-max",
		"qwen-long",
	}
}

func (a *QwenAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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

	var result openaiResponse
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

func (a *QwenAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	adapter := &OpenAIAdapter{
		apiKey:  a.apiKey,
		baseURL: a.baseURL,
		client:  a.client,
	}
	return adapter.ChatStream(ctx, req)
}

func (a *QwenAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}
