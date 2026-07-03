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

// OpenAIAdapter implements the Adapter interface for OpenAI.
type OpenAIAdapter struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter.
func NewOpenAIAdapter(cfg config.AIProviderConfig) *OpenAIAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	models := cfg.Models
	if len(models) == 0 {
		models = []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
		}
	}

	return &OpenAIAdapter{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for streaming
		},
	}
}

// NewOpenAIAdapterFromConfig creates an OpenAI adapter from explicit config values.
func NewOpenAIAdapterFromConfig(apiKey, baseURL string, models []string) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if len(models) == 0 {
		models = []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
		}
	}

	return &OpenAIAdapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (a *OpenAIAdapter) Provider() string {
	return "openai"
}

func (a *OpenAIAdapter) Models() []string {
	return a.models
}

func (a *OpenAIAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := a.buildRequest(req, false)

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

	choice := result.Choices[0]
	return &ChatResponse{
		ID:      result.ID,
		Content: choice.Message.Content,
		Model:   result.Model,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		FinishReason: choice.FinishReason,
	}, nil
}

func (a *OpenAIAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	body := a.buildRequest(req, true)

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
			if data == "[DONE]" {
				return
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				logger.Log.Debug("Failed to parse stream chunk", zap.Error(err))
				continue
			}

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				delta := choice.Delta
				finishReason := choice.FinishReason

				// Stream reasoning content
				if delta.ReasoningContent != "" {
					ch <- ChatStreamChunk{
						ID:     chunk.ID,
						Delta:  delta.ReasoningContent,
						Model:  chunk.Model,
						Type:   "reasoning",
					}
				}

				// Stream content (with finish reason if present)
				if delta.Content != "" {
					ch <- ChatStreamChunk{
						ID:     chunk.ID,
						Delta:  delta.Content,
						Model:  chunk.Model,
						Finish: finishReason,
					}
				} else if finishReason != "" {
					// Send finish signal only if no content was sent
					ch <- ChatStreamChunk{
						ID:     chunk.ID,
						Model:  chunk.Model,
						Finish: finishReason,
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Log.Error("Stream read error", zap.Error(err))
		}
	}()

	return ch, nil
}

func (a *OpenAIAdapter) CountTokens(model string, messages []Message) (int, error) {
	// Rough estimation: ~4 chars per token
	totalChars := 0
	for _, msg := range messages {
		totalChars += GetContentLength(msg) + len(msg.Role) + 4 // overhead
	}
	return totalChars / 4, nil
}

func (a *OpenAIAdapter) buildRequest(req *ChatRequest, stream bool) map[string]interface{} {
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]interface{}, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			m["tool_calls"] = toolCalls
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		messages = append(messages, m)
	}

	body := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   stream,
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
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				},
			})
		}
		body["tools"] = tools
	}

	return body
}

// OpenAI API response types

type openaiResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string     `json:"content"`
			ReasoningContent string     `json:"reasoning_content"`
			ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openaiStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content           string `json:"content"`
			ReasoningContent  string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
