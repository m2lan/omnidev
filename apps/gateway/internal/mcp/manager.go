package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

// Manager aggregates tools from all configured MCP servers.
type Manager struct {
	cfg     config.MCPConfig
	clients []*Client
	tools   []Tool
	exposed map[string]registeredTool
}

type registeredTool struct {
	client     *Client
	remoteName string
}

// NewManager creates a manager for the configured servers.
func NewManager(cfg config.MCPConfig) *Manager {
	return &Manager{cfg: cfg, exposed: map[string]registeredTool{}}
}

// Start connects to every configured server and collects its tools.
// A failing server is logged and skipped so chat keeps working.
func (m *Manager) Start(ctx context.Context) error {
	if !m.cfg.Enabled {
		return nil
	}
	if len(m.cfg.Servers) == 0 {
		return fmt.Errorf("mcp is enabled but no servers are configured")
	}

	var firstErr error
	for _, serverCfg := range m.cfg.Servers {
		if serverCfg.URL == "" {
			logger.Log.Warn("Skipping MCP server without URL", zap.String("server", serverCfg.Name))
			continue
		}
		client := NewClient(serverCfg)
		initCtx, cancel := context.WithTimeout(ctx, client.http.Timeout)
		err := client.Initialize(initCtx)
		cancel()
		if err != nil {
			logger.Log.Warn("Failed to initialize MCP server",
				zap.String("server", serverCfg.Name), zap.String("url", serverCfg.URL), zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			logger.Log.Warn("Failed to list MCP tools",
				zap.String("server", serverCfg.Name), zap.Error(err))
			continue
		}

		m.clients = append(m.clients, client)
		for _, tool := range tools {
			m.register(client, tool)
		}
		logger.Log.Info("MCP server ready",
			zap.String("server", serverCfg.Name), zap.Int("tools", len(tools)))
	}
	if len(m.clients) == 0 {
		if firstErr != nil {
			return fmt.Errorf("no MCP servers available: %w", firstErr)
		}
		return fmt.Errorf("no MCP servers available")
	}
	return nil
}

func (m *Manager) register(client *Client, tool Tool) {
	name := strings.TrimSpace(tool.Name)
	if name == "" {
		return
	}
	exposed := name
	if _, exists := m.exposed[exposed]; exists {
		prefix := sanitizeName(client.Name())
		exposed = prefix + "__" + name
		for i := 2; ; i++ {
			if _, exists := m.exposed[exposed]; !exists {
				break
			}
			exposed = fmt.Sprintf("%s%d__%s", prefix, i, name)
		}
	}
	tool.Name = exposed
	m.tools = append(m.tools, tool)
	m.exposed[exposed] = registeredTool{client: client, remoteName: name}
}

// Tools returns the model-facing tool catalog.
func (m *Manager) Tools() []Tool {
	if m == nil {
		return nil
	}
	return m.tools
}

// Enabled reports whether any MCP tools are available.
func (m *Manager) Enabled() bool {
	return m != nil && len(m.tools) > 0
}

// Call invokes a tool by its exposed name.
func (m *Manager) Call(ctx context.Context, name, arguments string) (string, bool, error) {
	if m == nil {
		return "", true, fmt.Errorf("MCP is not configured")
	}
	reg, ok := m.exposed[name]
	if !ok {
		return "", true, fmt.Errorf("unknown MCP tool %q", name)
	}
	return reg.client.CallTool(ctx, reg.remoteName, arguments)
}

// SchemaJSON returns the tool input schema as a raw JSON object, defaulting to
// an empty object schema when the server omitted it.
func (t Tool) SchemaJSON() json.RawMessage {
	if len(t.InputSchema) == 0 || string(t.InputSchema) == "null" {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return t.InputSchema
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "mcp"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		return "mcp"
	}
	return out
}
