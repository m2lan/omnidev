package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

// AnthropicAdapter implements the Adapter interface for Anthropic.
type AnthropicAdapter struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// NewAnthropicAdapter creates a new Anthropic adapter.
func NewAnthropicAdapter(cfg config.AIProviderConfig) *AnthropicAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	models := cfg.Models
	if len(models) == 0 {
		models = []string{
			"claude-opus-4-8",
			"claude-sonnet-4-6",
			"claude-haiku-4-5-20251001",
		}
	}

	return &AnthropicAdapter{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// NewAnthropicAdapterFromConfig creates an Anthropic adapter from user config.
func NewAnthropicAdapterFromConfig(apiKey, baseURL string, models []string) *AnthropicAdapter {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if len(models) == 0 {
		models = []string{
			"claude-opus-4-8",
			"claude-sonnet-4-6",
			"claude-haiku-4-5-20251001",
		}
	}

	return &AnthropicAdapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (a *AnthropicAdapter) Provider() string {
	return "anthropic"
}

func (a *AnthropicAdapter) Models() []string {
	return a.models
}

func (a *AnthropicAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := a.buildRequest(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text content from content blocks
	var content string
	for _, block := range result.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &ChatResponse{
		ID:      result.ID,
		Content: content,
		Model:   result.Model,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
		FinishReason: result.StopReason,
	}, nil
}

func (a *AnthropicAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	body := a.buildRequest(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan ChatStreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		var messageID string
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				logger.Log.Debug("Failed to parse stream chunk", zap.Error(err))
				continue
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil {
					messageID = event.Message.ID
				}
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Type == "text_delta" {
					ch <- ChatStreamChunk{
						ID:    messageID,
						Delta: event.Delta.Text,
					}
				}
			case "message_stop":
				ch <- ChatStreamChunk{
					ID:     messageID,
					Finish: "stop",
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Log.Error("Stream read error", zap.Error(err))
		}
	}()

	return ch, nil
}

func (a *AnthropicAdapter) CountTokens(model string, messages []Message) (int, error) {
	// Rough estimation: ~4 chars per token
	totalChars := 0
	for _, msg := range messages {
		totalChars += GetContentLength(msg) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}

func (a *AnthropicAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

func (a *AnthropicAdapter) buildRequest(req *ChatRequest, stream bool) map[string]interface{} {
	// Anthropic uses a separate system field, not a system message
	var systemPrompt string
	messages := make([]map[string]interface{}, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = GetTextContent(msg)
			continue
		}
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": 4096, // Required by Anthropic
	}

	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	if stream {
		body["stream"] = true
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}

	return body
}

// Anthropic API response types

type anthropicResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Message *struct {
		ID    string `json:"id"`
		Model string `json:"model"`
	} `json:"message,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
}
