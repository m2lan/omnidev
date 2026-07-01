// Package main is the entry point for the Workflow Service.
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

	"github.com/omnidev/services/workflow/internal/engine"
	"github.com/omnidev/services/workflow/internal/handler"
	"github.com/omnidev/services/workflow/internal/nodes"
	"github.com/omnidev/services/workflow/internal/repository"
	"github.com/omnidev/services/workflow/internal/service"
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

	logger.Log.Info("Starting Workflow Service", zap.String("version", version), zap.String("env", cfg.App.Env))

	ctx := context.Background()

	// Telemetry
	shutdownTel, err := telemetry.Init(ctx, "workflow-service", version, cfg.Telemetry.OTLPEndpoint)
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

	// Node Registry
	nodeRegistry := nodes.NewRegistry()
	nodeRegistry.Register(nodes.NewAINode(cfg.AI))
	nodeRegistry.Register(nodes.NewHTTPNode())
	nodeRegistry.Register(nodes.NewCodeNode())
	nodeRegistry.Register(nodes.NewConditionNode())
	nodeRegistry.Register(nodes.NewTransformNode())
	nodeRegistry.Register(nodes.NewDelayNode())

	// Workflow Engine
	wfEngine := engine.NewEngine(nodeRegistry)

	// Repositories
	wfRepo := repository.NewWorkflowRepository(db.Pool)
	runRepo := repository.NewRunRepository(db.Pool)
	nodeRunRepo := repository.NewNodeRunRepository(db.Pool)

	// Services
	wfSvc := service.NewWorkflowService(wfRepo, runRepo, nodeRunRepo, wfEngine)

	// Handlers
	wfHandler := handler.NewWorkflowHandler(wfSvc)

	// Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:9090"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "workflow"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		wf := v1.Group("/workflows")
		{
			wf.GET("", wfHandler.ListWorkflows)
			wf.POST("", wfHandler.CreateWorkflow)
			wf.GET("/:id", wfHandler.GetWorkflow)
			wf.PATCH("/:id", wfHandler.UpdateWorkflow)
			wf.DELETE("/:id", wfHandler.DeleteWorkflow)
			wf.POST("/:id/run", wfHandler.RunWorkflow)
			wf.GET("/:id/runs", wfHandler.ListRuns)
			wf.GET("/runs/:run_id", wfHandler.GetRun)
			wf.POST("/runs/:run_id/cancel", wfHandler.CancelRun)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+5) // Workflow on 8086
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("Workflow Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Workflow Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("Workflow Service exited")
}
