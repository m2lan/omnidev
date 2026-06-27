// Package main is the entry point for the API Gateway.
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
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/middleware"
	"github.com/omnidev/go-common/telemetry"

	"github.com/omnidev/gateway/internal/handler"
	"github.com/omnidev/gateway/internal/router"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:  cfg.App.LogLevel,
		Format: "console",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("Starting API Gateway",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_time", buildTime),
		zap.String("env", cfg.App.Env),
	)

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "gateway", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Failed to init telemetry, continuing without it", zap.Error(err))
	} else {
		defer func() {
			if err := shutdownTelemetry(ctx); err != nil {
				logger.Log.Error("Failed to shutdown telemetry", zap.Error(err))
			}
		}()
	}

	// Initialize Redis (optional, graceful degradation)
	var redisClient *cache.Redis
	redisClient, err = cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Warn("Failed to connect to Redis, rate limiting disabled", zap.Error(err))
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
	)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(version, commit, buildTime)

	// Setup Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	// Rate limiter (if Redis is available)
	if redisClient != nil {
		rl := middleware.NewRateLimiter(30, 50)
		r.Use(rl.RateLimit())
	}

	// Setup routes
	router.Setup(r, jwtManager, healthHandler)

	// HTTP server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // Long for SSE streams
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Log.Info("Server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	if redisClient != nil {
		redisClient.Close()
	}

	logger.Log.Info("Server exited")
}
