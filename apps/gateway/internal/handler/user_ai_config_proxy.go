package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"
)

// UserAIConfigProxyHandler proxies user AI config requests to the User Service.
type UserAIConfigProxyHandler struct {
	userServiceURL string
	client         *http.Client
}

// NewUserAIConfigProxyHandler creates a new proxy handler.
func NewUserAIConfigProxyHandler(userServiceURL string) *UserAIConfigProxyHandler {
	return &UserAIConfigProxyHandler{
		userServiceURL: userServiceURL,
		client:         &http.Client{},
	}
}

// Create proxies POST /api/v1/user/ai-configs
func (h *UserAIConfigProxyHandler) Create(c *gin.Context) {
	h.proxy(c, "POST", "/api/v1/user/ai-configs")
}

// List proxies GET /api/v1/user/ai-configs
func (h *UserAIConfigProxyHandler) List(c *gin.Context) {
	h.proxy(c, "GET", "/api/v1/user/ai-configs")
}

// Get proxies GET /api/v1/user/ai-configs/:id
func (h *UserAIConfigProxyHandler) Get(c *gin.Context) {
	h.proxy(c, "GET", fmt.Sprintf("/api/v1/user/ai-configs/%s", c.Param("id")))
}

// Update proxies PUT /api/v1/user/ai-configs/:id
func (h *UserAIConfigProxyHandler) Update(c *gin.Context) {
	h.proxy(c, "PUT", fmt.Sprintf("/api/v1/user/ai-configs/%s", c.Param("id")))
}

// Delete proxies DELETE /api/v1/user/ai-configs/:id
func (h *UserAIConfigProxyHandler) Delete(c *gin.Context) {
	h.proxy(c, "DELETE", fmt.Sprintf("/api/v1/user/ai-configs/%s", c.Param("id")))
}

// SetDefault proxies PUT /api/v1/user/ai-configs/:id/default
func (h *UserAIConfigProxyHandler) SetDefault(c *gin.Context) {
	h.proxy(c, "PUT", fmt.Sprintf("/api/v1/user/ai-configs/%s/default", c.Param("id")))
}

// TestConnection proxies POST /api/v1/user/ai-configs/:id/test
func (h *UserAIConfigProxyHandler) TestConnection(c *gin.Context) {
	h.proxy(c, "POST", fmt.Sprintf("/api/v1/user/ai-configs/%s/test", c.Param("id")))
}

// proxy forwards the request to the user service.
func (h *UserAIConfigProxyHandler) proxy(c *gin.Context, method, path string) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	url := h.userServiceURL + path

	var body io.Reader
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": 400, "message": "failed to read request body"},
			})
			return
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), method, url, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to create proxy request"},
		})
		return
	}

	// Forward headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	// Forward query parameters
	req.URL.RawQuery = c.Request.URL.RawQuery

	resp, err := h.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": 502, "message": "user service unavailable"},
		})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": 500, "message": "failed to read response"},
		})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", respBody)
}

// GetUserAIConfigForModel returns a user's AI config for a specific model.
// This is used internally by the gateway's chat service.
func (h *UserAIConfigProxyHandler) GetUserAIConfigForModel(userID uuid.UUID, modelID string) (*UserAIConfigResponse, error) {
	url := fmt.Sprintf("%s/api/v1/user/ai-configs/by-model/%s", h.userServiceURL, modelID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-User-ID", userID.String())

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var result struct {
		Data *UserAIConfigResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// UserAIConfigResponse represents a user AI config from the user service.
type UserAIConfigResponse struct {
	ID       uuid.UUID `json:"id"`
	Provider string    `json:"provider"`
	APIKey   string    `json:"api_key"` // Encrypted
	BaseURL  string    `json:"base_url"`
	Protocol string    `json:"protocol"`
	Models   []string  `json:"models"`
}
