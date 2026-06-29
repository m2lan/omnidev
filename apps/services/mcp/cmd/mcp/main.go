// Package main is the entry point for the MCP Service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/database"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/middleware"
	"github.com/omnidev/go-common/telemetry"

	"github.com/omnidev/services/mcp/internal/builtin"
	"github.com/omnidev/services/mcp/internal/handler"
	"github.com/omnidev/services/mcp/internal/repository"
	"github.com/omnidev/services/mcp/internal/service"
	"github.com/omnidev/services/mcp/internal/transport"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(logger.Config{Level: cfg.App.LogLevel, Format: "console"}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("Starting MCP Service", zap.String("version", version), zap.String("env", cfg.App.Env))

	ctx := context.Background()

	// Telemetry
	shutdownTel, err := telemetry.Init(ctx, "mcp-service", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Telemetry init failed", zap.Error(err))
	} else {
		defer func() { _ = shutdownTel(ctx) }()
	}

	// Database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		logger.Log.Fatal("DB connect failed", zap.Error(err))
	}
	defer db.Close()

	// Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Redis connect failed", zap.Error(err))
	}
	defer redisClient.Close()

	// JWT
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)

	// Built-in MCP Servers
	builtinRegistry := builtin.NewRegistry()
	builtinRegistry.Register(builtin.NewFilesystemServer())
	builtinRegistry.Register(builtin.NewGitHubServer())
	builtinRegistry.Register(builtin.NewSQLServer(db.Pool))
	builtinRegistry.Register(builtin.NewDockerServer())
	builtinRegistry.Register(builtin.NewBrowserServer())

	// Transport
	sseTransport := transport.NewSSETransport()

	// Repositories
	serverRepo := repository.NewServerRepository(db.Pool)
	toolRepo := repository.NewToolRepository(db.Pool)

	// Services
	mcpSvc := service.NewMCPService(serverRepo, toolRepo, builtinRegistry, sseTransport)

	// Handlers
	mcpHandler := handler.NewMCPHandler(mcpSvc, sseTransport)

	// Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mcp"})
	})

	// MCP SSE endpoint
	r.GET("/mcp/sse", mcpHandler.HandleSSE)
	r.POST("/mcp/message", mcpHandler.HandleMessage)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		mcp := v1.Group("/mcp")
		{
			mcp.GET("/servers", mcpHandler.ListServers)
			mcp.POST("/servers", mcpHandler.AddServer)
			mcp.GET("/servers/:id", mcpHandler.GetServer)
			mcp.PATCH("/servers/:id", mcpHandler.UpdateServer)
			mcp.DELETE("/servers/:id", mcpHandler.DeleteServer)
			mcp.GET("/servers/:id/tools", mcpHandler.ListTools)
			mcp.POST("/tools/:id/call", mcpHandler.CallTool)
			mcp.GET("/builtin", mcpHandler.ListBuiltinServers)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+4) // MCP on 8085
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("MCP Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down MCP Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("MCP Service exited")
}
