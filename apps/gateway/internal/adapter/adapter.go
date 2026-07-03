// Package adapter provides unified AI model adapters.
package adapter

import (
	"context"
	"fmt"
	"io"
)

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
	Tools       []Tool    `json:"tools,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentPart
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ContentPart represents a part of multimodal content.
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for type="text"
	ImageURL *ImageURL `json:"image_url,omitempty"` // for type="image_url"
}

// ImageURL represents an image URL with detail level.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// NewTextMessage creates a simple text message.
func NewTextMessage(role, content string) Message {
	return Message{Role: role, Content: content}
}

// NewMultimodalMessage creates a multimodal message with text and images.
func NewMultimodalMessage(role, text string, imageURLs []string) Message {
	parts := []ContentPart{{Type: "text", Text: text}}
	for _, url := range imageURLs {
		parts = append(parts, ContentPart{
			Type:     "image_url",
			ImageURL: &ImageURL{URL: url, Detail: "auto"},
		})
	}
	return Message{Role: role, Content: parts}
}

// GetTextContent extracts text content from a message (handles both string and multimodal).
func GetTextContent(msg Message) string {
	switch v := msg.Content.(type) {
	case string:
		return v
	case []ContentPart:
		for _, part := range v {
			if part.Type == "text" {
				return part.Text
			}
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetContentLength returns the approximate character length of message content.
func GetContentLength(msg Message) int {
	return len(GetTextContent(msg))
}

// Tool represents a function tool.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a function definition.
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID           string     `json:"id"`
	Content      string     `json:"content"`
	Model        string     `json:"model"`
	Usage        Usage      `json:"usage"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStreamChunk represents a streaming chunk.
type ChatStreamChunk struct {
	ID     string `json:"id"`
	Delta  string `json:"delta"`
	Model  string `json:"model"`
	Type   string `json:"type,omitempty"` // "reasoning" or empty for content
	Finish string `json:"finish,omitempty"`
	Usage  *Usage `json:"usage,omitempty"`
}

// Adapter defines the interface for AI model providers.
type Adapter interface {
	// Provider returns the provider name (e.g., "openai", "anthropic").
	Provider() string

	// Models returns the list of supported model IDs.
	Models() []string

	// Chat sends a chat completion request and returns a response.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream sends a chat completion request and returns a stream.
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamChunk, error)

	// CountTokens estimates the token count for a message.
	CountTokens(model string, messages []Message) (int, error)
}

// Registry manages multiple AI adapters.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register registers an adapter.
func (r *Registry) Register(adapter Adapter) {
	r.adapters[adapter.Provider()] = adapter
}

// Get returns an adapter by provider name.
func (r *Registry) Get(provider string) (Adapter, error) {
	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", provider)
	}
	return adapter, nil
}

// GetForModel returns the adapter that supports the given model.
func (r *Registry) GetForModel(modelID string) (Adapter, error) {
	for _, adapter := range r.adapters {
		for _, m := range adapter.Models() {
			if m == modelID {
				return adapter, nil
			}
		}
	}
	return nil, fmt.Errorf("no adapter found for model: %s", modelID)
}

// Providers returns all registered provider names.
func (r *Registry) Providers() []string {
	providers := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		providers = append(providers, name)
	}
	return providers
}

// StreamToString reads a stream channel and returns the complete response.
func StreamToString(ch <-chan ChatStreamChunk) (string, Usage, error) {
	var content string
	var usage Usage

	for chunk := range ch {
		content += chunk.Delta
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
	}

	return content, usage, nil
}

// ReadSSE reads Server-Sent Events from a reader and sends chunks to a channel.
func ReadSSE(reader io.Reader, ch chan<- ChatStreamChunk) error {
	// SSE reading logic would go here
	// For now, this is a placeholder
	return nil
}
