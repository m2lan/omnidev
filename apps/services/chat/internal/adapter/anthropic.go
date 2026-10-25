package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/omnidev/go-common/config"
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
			Timeout: 10 * time.Minute, // Long timeout for streaming
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
	var toolCalls []ToolCall
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			args := "{}"
			if len(block.Input) > 0 && string(block.Input) != "null" {
				args = string(block.Input)
			}
			tc := ToolCall{ID: block.ID, Type: "function"}
			tc.Function.Name = block.Name
			tc.Function.Arguments = args
			toolCalls = append(toolCalls, tc)
		}
	}

	finishReason := result.StopReason
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	return &ChatResponse{
		ID:        result.ID,
		Content:   content,
		Model:     result.Model,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
		FinishReason: finishReason,
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

		toolBlocks := map[int]*ToolCall{}
		toolArgs := map[int]*strings.Builder{}

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
			case "content_block_start":
				if event.ContentBlock.Type == "tool_use" {
					tc := &ToolCall{ID: event.ContentBlock.ID, Type: "function"}
					tc.Function.Name = event.ContentBlock.Name
					toolBlocks[event.Index] = tc
					toolArgs[event.Index] = &strings.Builder{}
				}
			case "content_block_delta":
				switch event.Delta.Type {
				case "text_delta":
					ch <- ChatStreamChunk{Delta: event.Delta.Text}
				case "input_json_delta":
					if args, ok := toolArgs[event.Index]; ok {
						args.WriteString(event.Delta.PartialJSON)
					}
				}
			case "message_stop":
				if len(toolBlocks) > 0 {
					indexes := make([]int, 0, len(toolBlocks))
					for index := range toolBlocks {
						indexes = append(indexes, index)
					}
					sort.Ints(indexes)

					calls := make([]ToolCall, 0, len(indexes))
					for _, index := range indexes {
						block := toolBlocks[index]
						args := toolArgs[index].String()
						if args == "" {
							args = "{}"
						}
						block.Function.Arguments = args
						calls = append(calls, *block)
					}
					ch <- ChatStreamChunk{Finish: "tool_calls", ToolCalls: calls}
				} else {
					ch <- ChatStreamChunk{Finish: "stop"}
				}
			}
		}
	}()

	return ch, nil
}

func (a *AnthropicAdapter) CountTokens(model string, messages []Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(GetTextContent(msg)) + len(msg.Role) + 4
	}
	return totalChars / 4, nil
}

func (a *AnthropicAdapter) buildRequest(req *ChatRequest, stream bool) map[string]interface{} {
	var systemParts []string
	messages := make([]map[string]interface{}, 0, len(req.Messages))

	appendBlocks := func(role string, blocks []map[string]interface{}) {
		if len(blocks) == 0 {
			return
		}
		if n := len(messages); n > 0 && messages[n-1]["role"] == role {
			prev, _ := messages[n-1]["content"].([]map[string]interface{})
			messages[n-1]["content"] = append(prev, blocks...)
			return
		}
		messages = append(messages, map[string]interface{}{"role": role, "content": blocks})
	}

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if text := GetTextContent(msg); text != "" {
				systemParts = append(systemParts, text)
			}
		case "tool":
			appendBlocks("user", []map[string]interface{}{{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"content":     GetTextContent(msg),
			}})
		default:
			blocks := anthropicContentBlocks(msg.Content)
			if msg.Role == "assistant" {
				for _, tc := range msg.ToolCalls {
					input := json.RawMessage("{}")
					if json.Valid([]byte(tc.Function.Arguments)) && tc.Function.Arguments != "" {
						input = json.RawMessage(tc.Function.Arguments)
					}
					blocks = append(blocks, map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": input,
					})
				}
			}
			appendBlocks(msg.Role, blocks)
		}
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": 4096,
		"stream":     stream,
	}

	if len(systemParts) > 0 {
		body["system"] = strings.Join(systemParts, "\n\n")
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, tool := range req.Tools {
			schema := tool.Function.Parameters
			if schema == nil {
				schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			tools = append(tools, map[string]interface{}{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": schema,
			})
		}
		body["tools"] = tools
	}

	return body
}

func anthropicContentBlocks(content interface{}) []map[string]interface{} {
	switch v := content.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []map[string]interface{}{{"type": "text", "text": v}}
	case []ContentPart:
		blocks := make([]map[string]interface{}, 0, len(v))
		for _, part := range v {
			switch part.Type {
			case "text":
				if part.Text != "" {
					blocks = append(blocks, map[string]interface{}{"type": "text", "text": part.Text})
				}
			case "image_url":
				if part.ImageURL == nil || part.ImageURL.URL == "" {
					continue
				}
				url := part.ImageURL.URL
				if strings.HasPrefix(url, "data:") {
					mediaType, data, ok := splitDataURL(url)
					if !ok {
						continue
					}
					blocks = append(blocks, map[string]interface{}{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": mediaType,
							"data":       data,
						},
					})
				} else {
					blocks = append(blocks, map[string]interface{}{
						"type":   "image",
						"source": map[string]interface{}{"type": "url", "url": url},
					})
				}
			}
		}
		return blocks
	default:
		return nil
	}
}

func splitDataURL(url string) (mediaType, data string, ok bool) {
	rest := strings.TrimPrefix(url, "data:")
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return "", "", false
	}
	header := rest[:comma]
	data = rest[comma+1:]
	mediaType = strings.TrimSuffix(header, ";base64")
	if mediaType == "" || data == "" {
		return "", "", false
	}
	return mediaType, data, true
}

// Anthropic API response types

type anthropicResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
}
