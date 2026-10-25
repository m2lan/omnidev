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

// OllamaAdapter implements the Adapter interface for local Ollama.
type OllamaAdapter struct {
	baseURL string
	models  []string
	client  *http.Client
}

// NewOllamaAdapter creates a new Ollama adapter.
func NewOllamaAdapter(cfg config.AIProviderConfig) *OllamaAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	models := cfg.Models
	if len(models) == 0 {
		models = []string{
			"llama3.1",
			"llama3",
			"mistral",
			"codellama",
			"phi3",
			"gemma2",
			"qwen2",
		}
	}

	return &OllamaAdapter{
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (a *OllamaAdapter) Provider() string {
	return "ollama"
}

func (a *OllamaAdapter) Models() []string {
	return a.models
}

func (a *OllamaAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := buildOllamaRequestBody(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	toolCalls := toToolCalls(result.Message.ToolCalls)
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	return &ChatResponse{
		ID:        fmt.Sprintf("ollama-%d", time.Now().UnixNano()),
		Content:   result.Message.Content,
		Model:     result.Model,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     result.PromptEvalCount,
			CompletionTokens: result.EvalCount,
			TotalTokens:      result.PromptEvalCount + result.EvalCount,
		},
		FinishReason: finishReason,
	}, nil
}

func (a *OllamaAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	body := buildOllamaRequestBody(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaStreamChunk
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					// Log error
				}
				return
			}

			if chunk.Message.Content != "" {
				ch <- ChatStreamChunk{
					Delta: chunk.Message.Content,
					Model: chunk.Model,
				}
			}

			if chunk.Done {
				calls := toToolCalls(chunk.Message.ToolCalls)
				if len(calls) > 0 {
					ch <- ChatStreamChunk{Model: chunk.Model, Finish: "tool_calls", ToolCalls: calls}
				} else {
					ch <- ChatStreamChunk{Model: chunk.Model, Finish: "stop"}
				}
				return
			}
		}
	}()

	return ch, nil
}

func (a *OllamaAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(GetTextContent(msg)) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}

func buildOllamaRequestBody(req *ChatRequest, stream bool) map[string]interface{} {
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": buildOllamaMessages(req.Messages),
		"stream":   stream,
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
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

func buildOllamaMessages(messages []Message) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			calls := make([]map[string]interface{}, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				args := interface{}(map[string]interface{}{})
				if json.Valid([]byte(tc.Function.Arguments)) && tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				calls = append(calls, map[string]interface{}{
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": args,
					},
				})
			}
			m["tool_calls"] = calls
		}
		out = append(out, m)
	}
	return out
}

func toToolCalls(raw []ollamaToolCall) []ToolCall {
	if len(raw) == 0 {
		return nil
	}
	calls := make([]ToolCall, 0, len(raw))
	for i, tc := range raw {
		args := "{}"
		if len(tc.Function.Arguments) > 0 && string(tc.Function.Arguments) != "null" {
			args = string(tc.Function.Arguments)
		}
		id := tc.ID
		if id == "" {
			id = fmt.Sprintf("ollama_call_%d_%d", time.Now().UnixNano(), i)
		}
		out := ToolCall{ID: id, Type: "function"}
		out.Function.Name = tc.Function.Name
		out.Function.Arguments = args
		calls = append(calls, out)
	}
	return calls
}

// Ollama API response types

type ollamaResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls"`
	} `json:"message"`
	Done            bool `json:"done"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	EvalCount       int  `json:"eval_count"`
}

type ollamaStreamChunk struct {
	Model   string `json:"model"`
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls"`
	} `json:"message"`
	Done bool `json:"done"`
}

type ollamaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}
