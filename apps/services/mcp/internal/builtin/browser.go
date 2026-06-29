package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// BrowserServer provides web browsing operations via MCP.
type BrowserServer struct {
	client *http.Client
}

func NewBrowserServer() *BrowserServer {
	return &BrowserServer{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *BrowserServer) Name() string       { return "browser" }
func (s *BrowserServer) Description() string { return "Web browsing: fetch pages, extract text" }

func (s *BrowserServer) Tools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "browser_fetch",
			Description: "Fetch a web page and return its content",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":     map[string]interface{}{"type": "string", "description": "URL to fetch"},
					"extract": map[string]interface{}{"type": "string", "enum": []string{"text", "html"}, "description": "Content extraction mode"},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "browser_search",
			Description: "Search the web (returns placeholder - integrate with search API)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query"},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (s *BrowserServer) HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	switch params.Name {
	case "browser_fetch":
		return s.fetchPage(params.Arguments)
	case "browser_search":
		return s.search(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *BrowserServer) fetchPage(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return toolError("url is required"), nil
	}

	extract := "text"
	if e, ok := args["extract"].(string); ok {
		extract = e
	}

	resp, err := s.client.Get(url)
	if err != nil {
		return toolError(fmt.Sprintf("failed to fetch URL: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolError(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	content := string(body)
	if extract == "text" {
		content = stripHTML(content)
		// Truncate to 10KB
		if len(content) > 10000 {
			content = content[:10000] + "\n... (truncated)"
		}
	}

	return toolSuccess(content), nil
}

func (s *BrowserServer) search(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return toolError("query is required"), nil
	}

	return toolSuccess(fmt.Sprintf("Web search not configured. Query: %s\nIntegrate with a search API (Google, Bing, etc.) to enable.", query)), nil
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			result.WriteString(" ")
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
