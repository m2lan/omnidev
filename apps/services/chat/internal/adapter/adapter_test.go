package adapter_test

import (
	"context"
	"testing"

	"github.com/omnidev/services/chat/internal/adapter"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := adapter.NewRegistry()

	mock := &mockAdapter{provider: "test"}
	registry.Register(mock)

	got, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Provider() != "test" {
		t.Errorf("Get().Provider() = %s, want test", got.Provider())
	}

	_, err = registry.Get("unknown")
	if err == nil {
		t.Error("Get('unknown') should return error")
	}
}

func TestRegistry_GetForModel(t *testing.T) {
	registry := adapter.NewRegistry()
	mock := &mockAdapter{
		provider: "openai",
		models:   []string{"gpt-4o", "gpt-4o-mini"},
	}
	registry.Register(mock)

	got, err := registry.GetForModel("gpt-4o")
	if err != nil {
		t.Fatalf("GetForModel() error = %v", err)
	}
	if got.Provider() != "openai" {
		t.Errorf("GetForModel().Provider() = %s, want openai", got.Provider())
	}

	_, err = registry.GetForModel("unknown-model")
	if err == nil {
		t.Error("GetForModel('unknown-model') should return error")
	}
}

func TestRegistry_Providers(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(&mockAdapter{provider: "openai"})
	registry.Register(&mockAdapter{provider: "anthropic"})

	providers := registry.Providers()
	if len(providers) != 2 {
		t.Errorf("Providers() returned %d, want 2", len(providers))
	}
}

func TestStreamToString(t *testing.T) {
	ch := make(chan adapter.ChatStreamChunk, 3)
	ch <- adapter.ChatStreamChunk{Delta: "Hello"}
	ch <- adapter.ChatStreamChunk{Delta: " "}
	ch <- adapter.ChatStreamChunk{Delta: "World", Usage: &adapter.Usage{TotalTokens: 10}}
	close(ch)

	content, usage, err := adapter.StreamToString(ch)
	if err != nil {
		t.Fatalf("StreamToString() error = %v", err)
	}

	if content != "Hello World" {
		t.Errorf("content = %q, want %q", content, "Hello World")
	}
	if usage.TotalTokens != 10 {
		t.Errorf("usage.TotalTokens = %d, want 10", usage.TotalTokens)
	}
}

// mockAdapter implements adapter.Adapter for testing.
type mockAdapter struct {
	provider string
	models   []string
}

func (m *mockAdapter) Provider() string { return m.provider }
func (m *mockAdapter) Models() []string { return m.models }
func (m *mockAdapter) Chat(ctx context.Context, req *adapter.ChatRequest) (*adapter.ChatResponse, error) {
	return nil, nil
}
func (m *mockAdapter) ChatStream(ctx context.Context, req *adapter.ChatRequest) (<-chan adapter.ChatStreamChunk, error) {
	return nil, nil
}
func (m *mockAdapter) CountTokens(model string, messages []adapter.Message) (int, error) {
	return 0, nil
}
