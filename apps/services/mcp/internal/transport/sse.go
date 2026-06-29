// Package transport provides MCP transport implementations.
package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// SSETransport implements the MCP SSE transport.
type SSETransport struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID       string
	UserID   string
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Done     chan struct{}
	messages chan *protocol.JSONRPCResponse
}

// NewSSETransport creates a new SSE transport.
func NewSSETransport() *SSETransport {
	return &SSETransport{
		clients: make(map[string]*SSEClient),
	}
}

// AddClient adds a new SSE client.
func (t *SSETransport) AddClient(w http.ResponseWriter, userID string) (*SSEClient, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	client := &SSEClient{
		ID:       uuid.New().String(),
		UserID:   userID,
		Writer:   w,
		Flusher:  flusher,
		Done:     make(chan struct{}),
		messages: make(chan *protocol.JSONRPCResponse, 100),
	}

	t.mu.Lock()
	t.clients[client.ID] = client
	t.mu.Unlock()

	logger.Log.Info("SSE client connected", zap.String("client_id", client.ID))

	return client, nil
}

// RemoveClient removes an SSE client.
func (t *SSETransport) RemoveClient(clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if client, ok := t.clients[clientID]; ok {
		close(client.Done)
		delete(t.clients, clientID)
		logger.Log.Info("SSE client disconnected", zap.String("client_id", clientID))
	}
}

// Send sends a message to a specific client.
func (t *SSETransport) Send(clientID string, resp *protocol.JSONRPCResponse) error {
	t.mu.RLock()
	client, ok := t.clients[clientID]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	select {
	case client.messages <- resp:
		return nil
	default:
		return fmt.Errorf("client buffer full: %s", clientID)
	}
}

// Broadcast sends a message to all connected clients.
func (t *SSETransport) Broadcast(resp *protocol.JSONRPCResponse) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, client := range t.clients {
		select {
		case client.messages <- resp:
		default:
			logger.Log.Warn("Client buffer full, skipping", zap.String("client_id", client.ID))
		}
	}
}

// ServeSSE starts serving SSE events to a client.
func (t *SSETransport) ServeSSE(client *SSEClient) {
	defer t.RemoveClient(client.ID)

	// Set SSE headers
	client.Writer.Header().Set("Content-Type", "text/event-stream")
	client.Writer.Header().Set("Cache-Control", "no-cache")
	client.Writer.Header().Set("Connection", "keep-alive")
	client.Writer.Header().Set("X-Accel-Buffering", "no")

	// Send initial connection event
	t.sendEvent(client, "connected", map[string]string{"client_id": client.ID})
	client.Flusher.Flush()

	for {
		select {
		case <-client.Done:
			return
		case msg := <-client.messages:
			data, err := json.Marshal(msg)
			if err != nil {
				logger.Log.Error("Failed to marshal message", zap.Error(err))
				continue
			}
			t.sendEvent(client, "message", string(data))
			client.Flusher.Flush()
		}
	}
}

func (t *SSETransport) sendEvent(client *SSEClient, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(client.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
}

// ClientCount returns the number of connected clients.
func (t *SSETransport) ClientCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.clients)
}
