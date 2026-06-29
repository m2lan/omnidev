// Package handler provides HTTP handlers for the MCP Service.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/mcp/internal/protocol"
	"github.com/omnidev/services/mcp/internal/service"
	"github.com/omnidev/services/mcp/internal/transport"
)

// MCPHandler handles MCP endpoints.
type MCPHandler struct {
	mcpSvc    *service.MCPService
	sseTransp *transport.SSETransport
}

// NewMCPHandler creates a new MCP handler.
func NewMCPHandler(mcpSvc *service.MCPService, sseTransp *transport.SSETransport) *MCPHandler {
	return &MCPHandler{mcpSvc: mcpSvc, sseTransp: sseTransp}
}

// HandleSSE handles MCP SSE connections.
// GET /mcp/sse
func (h *MCPHandler) HandleSSE(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	client, err := h.sseTransp.AddClient(c.Writer, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create SSE connection"})
		return
	}

	// Send initialize response
	initResp := protocol.NewSuccessResponse(nil, protocol.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolsCapability{ListChanged: true},
		},
		ServerInfo: protocol.ServerInfo{
			Name:    "omnidev-mcp",
			Version: "0.1.0",
		},
	})
	_ = h.sseTransp.Send(client.ID, initResp)

	// Serve SSE
	h.sseTransp.ServeSSE(client)
}

// HandleMessage handles MCP JSON-RPC messages.
// POST /mcp/message
func (h *MCPHandler) HandleMessage(c *gin.Context) {
	var req protocol.JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, protocol.NewErrorResponse(nil, protocol.InvalidRequest, "invalid request", nil))
		return
	}

	switch req.Method {
	case protocol.MethodInitialize:
		c.JSON(http.StatusOK, protocol.NewSuccessResponse(req.ID, protocol.InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: protocol.ServerCapabilities{
				Tools: &protocol.ToolsCapability{ListChanged: true},
			},
			ServerInfo: protocol.ServerInfo{
				Name:    "omnidev-mcp",
				Version: "0.1.0",
			},
		}))

	case protocol.MethodToolsList:
		tools := h.mcpSvc.ListBuiltinTools()
		c.JSON(http.StatusOK, protocol.NewSuccessResponse(req.ID, protocol.ToolsListResult{Tools: tools}))

	case protocol.MethodToolsCall:
		var params protocol.ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			c.JSON(http.StatusOK, protocol.NewErrorResponse(req.ID, protocol.InvalidParams, "invalid params", nil))
			return
		}
		// Find and call tool
		result, err := h.callBuiltinTool(c.Request.Context(), &params)
		if err != nil {
			c.JSON(http.StatusOK, protocol.NewErrorResponse(req.ID, protocol.InternalError, err.Error(), nil))
			return
		}
		c.JSON(http.StatusOK, protocol.NewSuccessResponse(req.ID, result))

	case protocol.MethodPing:
		c.JSON(http.StatusOK, protocol.NewSuccessResponse(req.ID, map[string]string{}))

	default:
		c.JSON(http.StatusOK, protocol.NewErrorResponse(req.ID, protocol.MethodNotFound, "method not found: "+req.Method, nil))
	}
}

func (h *MCPHandler) callBuiltinTool(ctx interface{}, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	return h.mcpSvc.CallToolBuiltin(ctx, params)
}

// ListServers returns a paginated list of MCP servers.
// GET /api/v1/mcp/servers
func (h *MCPHandler) ListServers(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	servers, total, err := h.mcpSvc.ListServers(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": servers,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// AddServer adds a new MCP server.
// POST /api/v1/mcp/servers
func (h *MCPHandler) AddServer(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.AddServerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	server, err := h.mcpSvc.AddServer(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": server})
}

// GetServer returns a server by ID.
// GET /api/v1/mcp/servers/:id
func (h *MCPHandler) GetServer(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid server ID")
		return
	}

	server, err := h.mcpSvc.GetServer(c.Request.Context(), userID, serverID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": server})
}

// UpdateServer updates an MCP server.
// PATCH /api/v1/mcp/servers/:id
func (h *MCPHandler) UpdateServer(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": 501, "message": "not implemented"}})
}

// DeleteServer deletes an MCP server.
// DELETE /api/v1/mcp/servers/:id
func (h *MCPHandler) DeleteServer(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid server ID")
		return
	}

	if err := h.mcpSvc.DeleteServer(c.Request.Context(), userID, serverID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "server deleted"}})
}

// ListTools returns tools for a server.
// GET /api/v1/mcp/servers/:id/tools
func (h *MCPHandler) ListTools(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid server ID")
		return
	}

	tools, err := h.mcpSvc.ListTools(c.Request.Context(), serverID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tools})
}

// CallTool calls an MCP tool.
// POST /api/v1/mcp/tools/:id/call
func (h *MCPHandler) CallTool(c *gin.Context) {
	toolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		// Try as tool name (for built-in tools)
		toolID = uuid.Nil
	}

	var input struct {
		ToolName string                 `json:"tool_name"`
		Input    map[string]interface{} `json:"input"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	result, err := h.mcpSvc.CallTool(c.Request.Context(), toolID, input.Input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ListBuiltinServers returns all built-in MCP servers.
// GET /api/v1/mcp/builtin
func (h *MCPHandler) ListBuiltinServers(c *gin.Context) {
	servers := h.mcpSvc.ListBuiltinServers()
	c.JSON(http.StatusOK, gin.H{"data": servers})
}

// --- Helpers ---

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
}

func badRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": detail}})
}

func handleError(c *gin.Context, err error) {
	if e, ok := err.(*appErr.AppError); ok {
		c.JSON(e.Code, gin.H{"error": gin.H{"code": e.Code, "message": e.Message, "detail": e.Detail}})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": "internal server error"}})
}
