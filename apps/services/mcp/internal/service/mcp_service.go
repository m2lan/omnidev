// Package service contains the business logic for the MCP Service.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/mcp/internal/builtin"
	"github.com/omnidev/services/mcp/internal/domain"
	"github.com/omnidev/services/mcp/internal/protocol"
	"github.com/omnidev/services/mcp/internal/repository"
	"github.com/omnidev/services/mcp/internal/transport"
)

// MCPService handles MCP operations.
type MCPService struct {
	serverRepo    repository.ServerRepository
	toolRepo      repository.ToolRepository
	builtinReg    *builtin.Registry
	transport     *transport.SSETransport
}

// NewMCPService creates a new MCP service.
func NewMCPService(
	serverRepo repository.ServerRepository,
	toolRepo repository.ToolRepository,
	builtinReg *builtin.Registry,
	transport *transport.SSETransport,
) *MCPService {
	return &MCPService{
		serverRepo: serverRepo,
		toolRepo:   toolRepo,
		builtinReg: builtinReg,
		transport:  transport,
	}
}

// AddServerInput defines the input for adding an MCP server.
type AddServerInput struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Transport   string            `json:"transport" validate:"required,oneof=sse stdio"`
	Endpoint    string            `json:"endpoint"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
}

// AddServer adds a new MCP server.
func (s *MCPService) AddServer(ctx context.Context, userID uuid.UUID, input *AddServerInput) (*domain.MCPServer, error) {
	server := &domain.MCPServer{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      input.Name,
		Transport: domain.TransportType(input.Transport),
		IsBuiltin: false,
		IsActive:  true,
		Metadata:  map[string]interface{}{},
	}

	if input.Description != "" {
		server.Description = &input.Description
	}
	if input.Endpoint != "" {
		server.Endpoint = &input.Endpoint
	}
	if input.Command != "" {
		server.Command = &input.Command
	}
	server.Args = input.Args
	server.Env = input.Env

	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, appErr.Wrap(err, "failed to add server")
	}

	return server, nil
}

// GetServer returns a server by ID.
func (s *MCPService) GetServer(ctx context.Context, userID, serverID uuid.UUID) (*domain.MCPServer, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, appErr.NotFound("server")
	}
	if server.UserID != userID && !server.IsBuiltin {
		return nil, appErr.ErrForbidden
	}
	return server, nil
}

// ListServers returns a paginated list of servers.
func (s *MCPService) ListServers(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.MCPServer, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.serverRepo.List(ctx, userID, offset, pageSize)
}

// DeleteServer deletes an MCP server.
func (s *MCPService) DeleteServer(ctx context.Context, userID, serverID uuid.UUID) error {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return appErr.NotFound("server")
	}
	if server.UserID != userID {
		return appErr.ErrForbidden
	}
	if server.IsBuiltin {
		return appErr.Validation("cannot delete built-in server")
	}
	return s.serverRepo.Delete(ctx, serverID)
}

// ListBuiltinServers returns all built-in servers.
func (s *MCPService) ListBuiltinServers() []map[string]interface{} {
	servers := s.builtinReg.List()
	result := make([]map[string]interface{}, 0, len(servers))
	for _, srv := range servers {
		tools := srv.Tools()
		toolNames := make([]string, len(tools))
		for i, t := range tools {
			toolNames[i] = t.Name
		}
		result = append(result, map[string]interface{}{
			"name":        srv.Name(),
			"description": srv.Description(),
			"tools":       toolNames,
		})
	}
	return result
}

// CallTool calls an MCP tool.
func (s *MCPService) CallTool(ctx context.Context, toolID uuid.UUID, input map[string]interface{}) (*domain.ToolCallResponse, error) {
	// Try built-in tools first
	for _, srv := range s.builtinReg.List() {
		for _, t := range srv.Tools() {
			if t.Name == toolID.String() {
				start := time.Now()
				result, err := srv.HandleToolCall(ctx, &protocol.ToolCallParams{
					Name:      t.Name,
					Arguments: input,
				})
				latency := int(time.Since(start).Milliseconds())

				if err != nil {
					return &domain.ToolCallResponse{
						ToolID:   toolID,
						Error:    err.Error(),
						Duration: latency,
					}, nil
				}

				output := make(map[string]interface{})
				if result.Content != nil && len(result.Content) > 0 {
					output["text"] = result.Content[0].Text
				}
				output["is_error"] = result.IsError

				return &domain.ToolCallResponse{
					ToolID:   toolID,
					Output:   output,
					Duration: latency,
				}, nil
			}
		}
	}

	// Try database tools
	tool, err := s.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return nil, appErr.NotFound("tool")
	}

	logger.Log.Info("Calling MCP tool",
		zap.String("tool_id", tool.ID.String()),
		zap.String("tool_name", tool.Name),
	)

	// TODO: Call external MCP server
	return &domain.ToolCallResponse{
		ToolID: toolID,
		Error:  "external MCP server calls not yet implemented",
	}, nil
}

// ListTools returns tools for a server.
func (s *MCPService) ListTools(ctx context.Context, serverID uuid.UUID) ([]domain.MCPTool, error) {
	return s.toolRepo.ListByServer(ctx, serverID)
}
