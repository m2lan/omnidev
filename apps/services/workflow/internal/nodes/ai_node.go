package nodes

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

// AINode calls an AI model for text generation.
type AINode struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewAINode(cfg config.AIConfig) *AINode {
	baseURL := cfg.OpenAI.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &AINode{
		apiKey:  cfg.OpenAI.APIKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (n *AINode) Type() string        { return "ai" }
func (n *AINode) Name() string        { return "AI Model" }
func (n *AINode) Description() string { return "Call an AI model for text generation" }

func (n *AINode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"model":       map[string]interface{}{"type": "string", "description": "Model ID", "default": "gpt-4o-mini"},
			"prompt":      map[string]interface{}{"type": "string", "description": "The prompt text"},
			"system":      map[string]interface{}{"type": "string", "description": "System prompt"},
			"temperature": map[string]interface{}{"type": "number", "description": "Temperature", "default": 0.7},
			"max_tokens":  map[string]interface{}{"type": "number", "description": "Max tokens", "default": 1000},
		},
		"required": []string{"prompt"},
	}
}

func (n *AINode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	model, _ := nodeCtx.Input["model"].(string)
	if model == "" {
		model = "gpt-4o-mini"
	}
	prompt, _ := nodeCtx.Input["prompt"].(string)
	system, _ := nodeCtx.Input["system"].(string)
	temperature := 0.7
	if t, ok := nodeCtx.Input["temperature"].(float64); ok {
		temperature = t
	}
	maxTokens := 1000
	if m, ok := nodeCtx.Input["max_tokens"].(float64); ok {
		maxTokens = int(m)
	}

	messages := []map[string]string{}
	if system != "" {
		messages = append(messages, map[string]string{"role": "system", "content": system})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	body := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", n.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+n.apiKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &NodeOutput{Status: "failed", Error: fmt.Sprintf("API error: %s", string(bodyBytes))}, nil
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}

	output := ""
	if len(result.Choices) > 0 {
		output = result.Choices[0].Message.Content
	}

	return &NodeOutput{
		Status: "success",
		Data: map[string]interface{}{
			"output":       output,
			"model":        model,
			"total_tokens": result.Usage.TotalTokens,
		},
	}, nil
}
