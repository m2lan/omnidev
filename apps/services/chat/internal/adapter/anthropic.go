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

// AnthropicAdapter implements the Adapter interface for Anthropic Claude.
type AnthropicAdapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicAdapter creates a new Anthropic adapter.
func NewAnthropicAdapter(cfg config.AIProviderConfig) *AnthropicAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &AnthropicAdapter{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (a *AnthropicAdapter) Provider() string {
	return "anthropic"
}

func (a *AnthropicAdapter) Models() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
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

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	content := ""
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

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

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

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					ch <- ChatStreamChunk{
						Delta: event.Delta.Text,
					}
				}
			case "message_stop":
				ch <- ChatStreamChunk{
					Finish: "stop",
				}
			}
		}
	}()

	return ch, nil
}

func (a *AnthropicAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}

func (a *AnthropicAdapter) buildRequest(req *ChatRequest, stream bool) map[string]interface{} {
	var system string
	messages := make([]map[string]interface{}, 0)

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			system = msg.Content
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
		"max_tokens": 4096,
		"stream":     stream,
	}

	if system != "" {
		body["system"] = system
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
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
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}
