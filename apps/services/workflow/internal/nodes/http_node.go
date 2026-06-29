package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPNode makes HTTP requests to external APIs.
type HTTPNode struct {
	client *http.Client
}

func NewHTTPNode() *HTTPNode {
	return &HTTPNode{client: &http.Client{Timeout: 30 * time.Second}}
}

func (n *HTTPNode) Type() string        { return "http" }
func (n *HTTPNode) Name() string        { return "HTTP Request" }
func (n *HTTPNode) Description() string { return "Make HTTP requests to external APIs" }

func (n *HTTPNode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"method":  map[string]interface{}{"type": "string", "enum": []string{"GET", "POST", "PUT", "PATCH", "DELETE"}, "default": "GET"},
			"url":     map[string]interface{}{"type": "string", "description": "Request URL"},
			"headers": map[string]interface{}{"type": "object", "description": "Request headers"},
			"body":    map[string]interface{}{"type": "object", "description": "Request body (for POST/PUT/PATCH)"},
		},
		"required": []string{"method", "url"},
	}
}

func (n *HTTPNode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	method, _ := nodeCtx.Input["method"].(string)
	url, _ := nodeCtx.Input["url"].(string)
	if url == "" {
		return &NodeOutput{Status: "failed", Error: "url is required"}, nil
	}

	var bodyReader io.Reader
	if body, ok := nodeCtx.Input["body"]; ok && body != nil {
		bodyData, err := json.Marshal(body)
		if err != nil {
			return &NodeOutput{Status: "failed", Error: err.Error()}, nil
		}
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}

	if headers, ok := nodeCtx.Input["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := n.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &NodeOutput{Status: "failed", Error: err.Error()}, nil
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(respBody, &jsonData); err != nil {
		jsonData = string(respBody)
	}

	return &NodeOutput{
		Status: "success",
		Data: map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        jsonData,
			"headers":     resp.Header,
			"latency_ms":  latency,
		},
	}, nil
}
