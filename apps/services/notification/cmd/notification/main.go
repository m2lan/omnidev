// Package main is the entry point for the Notification Service.
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

	"github.com/omnidev/services/notification/internal/channels"
	"github.com/omnidev/services/notification/internal/handler"
	"github.com/omnidev/services/notification/internal/repository"
	"github.com/omnidev/services/notification/internal/service"
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

	logger.Log.Info("Starting Notification Service", zap.String("version", version))

	ctx := context.Background()

	shutdownTel, err := telemetry.Init(ctx, "notification-service", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Telemetry init failed", zap.Error(err))
	} else {
		defer func() { _ = shutdownTel(ctx) }()
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		logger.Log.Fatal("DB connect failed", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Redis connect failed", zap.Error(err))
	}
	defer redisClient.Close()

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)

	// Channels
	channelRegistry := channels.NewRegistry()
	channelRegistry.Register(channels.NewInAppChannel())
	channelRegistry.Register(channels.NewEmailChannel())
	channelRegistry.Register(channels.NewSlackChannel())
	channelRegistry.Register(channels.NewWebhookChannel())

	// Repositories
	notifRepo := repository.NewNotificationRepository(db.Pool)

	// Services
	notifSvc := service.NewNotificationService(notifRepo, channelRegistry)

	// Handlers
	notifHandler := handler.NewNotificationHandler(notifSvc)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "notification"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		notif := v1.Group("/notifications")
		{
			notif.GET("", notifHandler.ListNotifications)
			notif.POST("/send", notifHandler.SendNotification)
			notif.PATCH("/:id/read", notifHandler.MarkAsRead)
			notif.POST("/read-all", notifHandler.MarkAllAsRead)
			notif.GET("/unread-count", notifHandler.UnreadCount)
			notif.GET("/preferences", notifHandler.GetPreferences)
			notif.PATCH("/preferences", notifHandler.UpdatePreferences)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+8)
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("Notification Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Notification Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("Notification Service exited")
}
