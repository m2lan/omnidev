package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

func TestMain(m *testing.M) {
	logger.InitDefault()
	m.Run()
}

type rpcFrame struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func writeJSONRPC(t *testing.T, w http.ResponseWriter, id json.RawMessage, result interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{"jsonrpc": "2.0", "id": json.RawMessage(id), "result": result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(config.MCPServerConfig{Name: "test", URL: srv.URL, TimeoutSeconds: 5}), srv
}

func TestClientInitializeAndListTools(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("MCP-Protocol-Version"); got != protocolVersion {
			t.Errorf("protocol header = %q, want %q", got, protocolVersion)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		var frame rpcFrame
		if err := json.Unmarshal(body, &frame); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if frame.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Mcp-Session-Id", "sess-1")
		switch frame.Method {
		case "initialize":
			writeJSONRPC(t, w, frame.ID, map[string]interface{}{
				"protocolVersion": protocolVersion,
				"serverInfo":      map[string]string{"name": "fake", "version": "1.0"},
			})
		case "tools/list":
			var params struct {
				Cursor string `json:"cursor"`
			}
			_ = json.Unmarshal(frame.Params, &params)
			if params.Cursor == "" {
				writeJSONRPC(t, w, frame.ID, map[string]interface{}{
					"tools": []map[string]interface{}{
						{"name": "search", "description": "Search", "inputSchema": map[string]interface{}{"type": "object"}},
					},
					"nextCursor": "page-2",
				})
			} else {
				writeJSONRPC(t, w, frame.ID, map[string]interface{}{
					"tools": []map[string]interface{}{
						{"name": "fetch", "inputSchema": map[string]interface{}{"type": "object"}},
					},
				})
			}
		default:
			t.Errorf("unexpected method %q", frame.Method)
		}
	})
	client.apiKey = "secret"

	if err := client.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if client.sessionID != "sess-1" {
		t.Fatalf("session id = %q", client.sessionID)
	}

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 2 || tools[0].Name != "search" || tools[1].Name != "fetch" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestClientCallToolJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var frame rpcFrame
		_ = json.Unmarshal(body, &frame)
		if frame.Method != "tools/call" {
			t.Errorf("method = %q", frame.Method)
		}
		writeJSONRPC(t, w, json.RawMessage(`7`), map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "first"}, {"type": "text", "text": "second"}},
		})
	})

	// The request id is generated internally, so accept whatever the client sends.
	out, isErr, err := client.CallTool(context.Background(), "search", `{"query":"x"}`)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if isErr {
		t.Fatal("unexpected tool error")
	}
	if out != "first\nsecond" {
		t.Fatalf("output = %q", out)
	}
}

func TestClientCallToolSSE(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var frame rpcFrame
		_ = json.Unmarshal(body, &frame)

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"streamed\"}]}}\n\n", frame.ID)
	})

	out, _, err := client.CallTool(context.Background(), "search", "{}")
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if out != "streamed" {
		t.Fatalf("output = %q", out)
	}
}

func TestClientCallToolError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var frame rpcFrame
		_ = json.Unmarshal(body, &frame)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(frame.ID),
			"error":   map[string]interface{}{"code": -32000, "message": "boom"},
		})
	})

	_, _, err := client.CallTool(context.Background(), "search", "{}")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientCallToolInvalidArguments(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for invalid arguments")
	})

	out, isErr, err := client.CallTool(context.Background(), "search", "not-json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isErr || out == "" {
		t.Fatalf("expected tool error output, got %q isErr=%v", out, isErr)
	}
}

func TestClientHTTPError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	})

	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestManagerCollisionAndCall(t *testing.T) {
	newServer := func(prefix string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var frame rpcFrame
			_ = json.Unmarshal(body, &frame)
			if frame.ID == nil {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			switch frame.Method {
			case "initialize":
				writeJSONRPC(t, w, frame.ID, map[string]interface{}{"protocolVersion": protocolVersion})
			case "tools/list":
				writeJSONRPC(t, w, frame.ID, map[string]interface{}{
					"tools": []map[string]interface{}{{"name": "search"}},
				})
			case "tools/call":
				writeJSONRPC(t, w, frame.ID, map[string]interface{}{
					"content": []map[string]string{{"type": "text", "text": prefix}},
				})
			}
		}))
	}

	first := newServer("first")
	defer first.Close()
	second := newServer("second")
	defer second.Close()

	manager := NewManager(config.MCPConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{Name: "alpha server", URL: first.URL},
			{Name: "beta", URL: second.URL},
		},
	})
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	names := make(map[string]bool)
	for _, tool := range manager.Tools() {
		names[tool.Name] = true
	}
	if !names["search"] || !names["beta__search"] {
		t.Fatalf("unexpected exposed names: %v", names)
	}

	out, _, err := manager.Call(context.Background(), "search", "{}")
	if err != nil || out != "first" {
		t.Fatalf("call search: out=%q err=%v", out, err)
	}

	out, _, err = manager.Call(context.Background(), "beta__search", "{}")
	if err != nil || out != "second" {
		t.Fatalf("call beta__search: out=%q err=%v", out, err)
	}

	if _, _, err := manager.Call(context.Background(), "missing", "{}"); err == nil {
		t.Fatal("expected unknown-tool error")
	}
}

func TestManagerStartSkipsUnreachableServer(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var frame rpcFrame
		_ = json.Unmarshal(body, &frame)
		if frame.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		switch frame.Method {
		case "initialize":
			writeJSONRPC(t, w, frame.ID, map[string]interface{}{"protocolVersion": protocolVersion})
		case "tools/list":
			writeJSONRPC(t, w, frame.ID, map[string]interface{}{
				"tools": []map[string]interface{}{{"name": "ping"}},
			})
		}
	}))
	defer good.Close()

	manager := NewManager(config.MCPConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{Name: "down", URL: "http://127.0.0.1:1/mcp", TimeoutSeconds: 1},
			{Name: "up", URL: good.URL},
		},
	})
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(manager.Tools()) != 1 || manager.Tools()[0].Name != "ping" {
		t.Fatalf("unexpected tools: %+v", manager.Tools())
	}
}

func TestSchemaJSONDefault(t *testing.T) {
	var tool Tool
	if got := string(tool.SchemaJSON()); got != `{"type":"object","properties":{}}` {
		t.Fatalf("schema = %s", got)
	}
}
