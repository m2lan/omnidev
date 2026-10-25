// Package mcp provides a minimal Model Context Protocol client for the chat
// service. It speaks the Streamable HTTP transport (JSON-RPC 2.0 over POST)
// and exposes remote tools to the model.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/omnidev/go-common/config"
)

const (
	protocolVersion = "2024-11-05"
	clientName      = "omnidev-chat"
	clientVersion   = "0.1.0"
)

// Tool is a tool advertised by an MCP server.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type toolsListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor"`
}

type toolCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	IsError           bool            `json:"isError"`
}

// Client talks to a single MCP server.
type Client struct {
	name      string
	url       string
	apiKey    string
	headers   map[string]string
	http      *http.Client
	requestID atomic.Int64
	sessionID string
	protocol  string
}

// NewClient creates a client for one MCP server.
func NewClient(cfg config.MCPServerConfig) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		name:     cfg.Name,
		url:      cfg.URL,
		apiKey:   cfg.APIKey,
		headers:  cfg.Headers,
		http:     &http.Client{Timeout: timeout},
		protocol: protocolVersion,
	}
}

// Name returns the configured server name.
func (c *Client) Name() string { return c.name }

// Initialize performs the MCP handshake.
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    clientName,
			"version": clientVersion,
		},
	}
	result, err := c.call(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	var init initializeResult
	if err := json.Unmarshal(result, &init); err != nil {
		return fmt.Errorf("decode initialize result: %w", err)
	}
	if init.ProtocolVersion != "" {
		c.protocol = init.ProtocolVersion
	}

	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}
	return nil
}

// ListTools fetches every tool page advertised by the server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var tools []Tool
	cursor := ""
	for page := 0; page < 20; page++ {
		params := map[string]interface{}{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := c.call(ctx, "tools/list", params)
		if err != nil {
			return nil, fmt.Errorf("tools/list: %w", err)
		}
		var result toolsListResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("decode tools/list: %w", err)
		}
		tools = append(tools, result.Tools...)
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}
	return tools, nil
}

// CallTool invokes a tool and returns its textual output.
func (c *Client) CallTool(ctx context.Context, name, arguments string) (string, bool, error) {
	if strings.TrimSpace(arguments) == "" {
		arguments = "{}"
	}
	if !json.Valid([]byte(arguments)) {
		return "invalid tool arguments: expected a JSON object", true, nil
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": json.RawMessage(arguments),
	}
	raw, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return "", false, fmt.Errorf("tools/call %s: %w", name, err)
	}
	var result toolCallResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", false, fmt.Errorf("decode tools/call %s: %w", name, err)
	}

	var parts []string
	for _, item := range result.Content {
		if item.Type == "text" && item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	if len(parts) == 0 && len(result.StructuredContent) > 0 && string(result.StructuredContent) != "null" {
		parts = append(parts, string(result.StructuredContent))
	}
	text := strings.Join(parts, "\n")
	if text == "" {
		text = "(no output)"
	}
	return text, result.IsError, nil
}

func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.requestID.Add(1)
	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("empty result")
	}
	return resp.Result, nil
}

func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	req := rpcRequest{JSONRPC: "2.0", Method: method, Params: params}
	_, err := c.do(ctx, req)
	return err
}

func (c *Client) do(ctx context.Context, payload rpcRequest) (*rpcResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", c.protocol)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.sessionID = sessionID
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if payload.ID == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, nil
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return readSSEResponse(resp.Body)
	}

	var result rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func readSSEResponse(body io.Reader) (*rpcResponse, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}
		if resp.Result != nil || resp.Error != nil {
			return &resp, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read event stream: %w", err)
	}
	return nil, fmt.Errorf("event stream ended without a response")
}
