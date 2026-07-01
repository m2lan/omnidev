// Package main is the entry point for the Agent Service.
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

	"github.com/omnidev/services/agent/internal/executor"
	"github.com/omnidev/services/agent/internal/handler"
	"github.com/omnidev/services/agent/internal/planner"
	"github.com/omnidev/services/agent/internal/repository"
	"github.com/omnidev/services/agent/internal/service"
	"github.com/omnidev/services/agent/internal/tools"
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

	logger.Log.Info("Starting Agent Service", zap.String("version", version), zap.String("env", cfg.App.Env))

	ctx := context.Background()

	// Telemetry
	shutdownTel, err := telemetry.Init(ctx, "agent-service", version, cfg.Telemetry.OTLPEndpoint)
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

	// Tool Registry
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewFileTool())
	toolRegistry.Register(tools.NewSearchTool())
	toolRegistry.Register(tools.NewCalculatorTool())
	toolRegistry.Register(tools.NewCodeExecTool())

	// Planner
	agentPlanner := planner.NewPlanner(cfg.AI)

	// Sandbox
	sandboxMgr := executor.NewSandboxManager(cfg.Sandbox)

	// Executor
	agentExecutor := executor.NewExecutor(toolRegistry, sandboxMgr, agentPlanner)

	// Repositories
	agentRepo := repository.NewAgentRepository(db.Pool)
	runRepo := repository.NewRunRepository(db.Pool)
	stepRepo := repository.NewStepRepository(db.Pool)

	// Services
	agentSvc := service.NewAgentService(agentRepo, runRepo, stepRepo, agentExecutor, agentPlanner)

	// Handlers
	agentHandler := handler.NewAgentHandler(agentSvc)

	// Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:9090"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "agent"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		agents := v1.Group("/agents")
		{
			agents.GET("", agentHandler.ListAgents)
			agents.POST("", agentHandler.CreateAgent)
			agents.GET("/:id", agentHandler.GetAgent)
			agents.PATCH("/:id", agentHandler.UpdateAgent)
			agents.DELETE("/:id", agentHandler.DeleteAgent)
			agents.POST("/:id/run", agentHandler.RunAgent)
			agents.GET("/:id/runs", agentHandler.ListRuns)
			agents.GET("/runs/:run_id", agentHandler.GetRun)
			agents.POST("/runs/:run_id/cancel", agentHandler.CancelRun)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+3) // Agent on 8084
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("Agent Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Agent Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("Agent Service exited")
}
