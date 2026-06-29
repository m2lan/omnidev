// Package main is the entry point for the Deploy Service.
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

	"github.com/omnidev/services/deploy/internal/builder"
	"github.com/omnidev/services/deploy/internal/handler"
	"github.com/omnidev/services/deploy/internal/platforms"
	"github.com/omnidev/services/deploy/internal/repository"
	"github.com/omnidev/services/deploy/internal/service"
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

	logger.Log.Info("Starting Deploy Service", zap.String("version", version), zap.String("env", cfg.App.Env))

	ctx := context.Background()

	// Telemetry
	shutdownTel, err := telemetry.Init(ctx, "deploy-service", version, cfg.Telemetry.OTLPEndpoint)
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

	// Builder
	imageBuilder := builder.NewDockerBuilder()

	// Platforms
	platformRegistry := platforms.NewRegistry()
	platformRegistry.Register(platforms.NewDockerPlatform())
	platformRegistry.Register(platforms.NewKubernetesPlatform(cfg))

	// Repositories
	deployRepo := repository.NewDeployRepository(db.Pool)

	// Services
	deploySvc := service.NewDeployService(deployRepo, imageBuilder, platformRegistry)

	// Handlers
	deployHandler := handler.NewDeployHandler(deploySvc)

	// Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "deploy"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		deploys := v1.Group("/deployments")
		{
			deploys.GET("", deployHandler.ListDeployments)
			deploys.POST("", deployHandler.CreateDeployment)
			deploys.GET("/:id", deployHandler.GetDeployment)
			deploys.POST("/:id/rollback", deployHandler.RollbackDeployment)
			deploys.GET("/:id/logs", deployHandler.GetDeploymentLogs)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+6) // Deploy on 8087
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("Deploy Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Deploy Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("Deploy Service exited")
}
