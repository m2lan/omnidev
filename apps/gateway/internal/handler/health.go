// Package handler provides HTTP handlers for the API Gateway.
package handler

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	version   string
	commit    string
	buildTime string
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(version, commit, buildTime string) *HealthHandler {
	return &HealthHandler{
		version:   version,
		commit:    commit,
		buildTime: buildTime,
	}
}

// Health returns a simple health check response.
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Ready returns readiness status.
func (h *HealthHandler) Ready(c *gin.Context) {
	// TODO: check database, redis, kafka connections
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Info returns application version information.
func (h *HealthHandler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    h.version,
		"commit":     h.commit,
		"build_time": h.buildTime,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"goroutines": runtime.NumGoroutine(),
	})
}
