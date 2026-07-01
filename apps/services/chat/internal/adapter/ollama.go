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
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}

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

	return &ChatResponse{
		ID:      fmt.Sprintf("ollama-%d", time.Now().UnixNano()),
		Content: result.Message.Content,
		Model:   result.Model,
		Usage: Usage{
			PromptTokens:     result.PromptEvalCount,
			CompletionTokens: result.EvalCount,
			TotalTokens:      result.PromptEvalCount + result.EvalCount,
		},
		FinishReason: "stop",
	}, nil
}

func (a *OllamaAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error) {
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

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

			ch <- ChatStreamChunk{
				Delta:  chunk.Message.Content,
				Model:  chunk.Model,
				Finish: boolToFinish(chunk.Done),
			}

			if chunk.Done {
				return
			}
		}
	}()

	return ch, nil
}

func (a *OllamaAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}

func boolToFinish(done bool) string {
	if done {
		return "stop"
	}
	return ""
}

// Ollama API response types

type ollamaResponse struct {
	Model              string `json:"model"`
	Message            struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done               bool `json:"done"`
	PromptEvalCount    int  `json:"prompt_eval_count"`
	EvalCount          int  `json:"eval_count"`
}

type ollamaStreamChunk struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}
