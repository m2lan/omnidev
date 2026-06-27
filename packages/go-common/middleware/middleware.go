package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

// RequestID adds a unique request ID to each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger logs HTTP requests.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", c.GetString("X-Request-ID")),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if userID, ok := GetUserID(c); ok {
			fields = append(fields, zap.String("user_id", userID.String()))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			logger.Log.Error("HTTP request", fields...)
		case status >= 400:
			logger.Log.Warn("HTTP request", fields...)
		default:
			logger.Log.Info("HTTP request", fields...)
		}
	}
}

// Recovery recovers from panics and returns 500.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Log.Error("Panic recovered",
					zap.String("request_id", c.GetString("X-Request-ID")),
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       500,
						"message":    "internal server error",
						"request_id": c.GetString("X-Request-ID"),
					},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORS adds Cross-Origin Resource Sharing headers.
func CORS(allowOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		for _, o := range allowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Total-Count")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiter implements a simple in-memory token bucket rate limiter.
// For production, use Redis-based rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int
	burst    int
	cleanup  time.Duration
}

type bucket struct {
	tokens    float64
	lastTime  time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rate int, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		cleanup: 1 * time.Minute,
	}

	// Cleanup old buckets periodically
	go func() {
		ticker := time.NewTicker(rl.cleanup)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.buckets {
				if now.Sub(b.lastTime) > rl.cleanup {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// RateLimit returns a middleware that rate limits requests by IP.
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		rl.mu.Lock()
		b, exists := rl.buckets[key]
		if !exists {
			b = &bucket{
				tokens:   float64(rl.burst),
				lastTime: time.Now(),
			}
			rl.buckets[key] = b
		}

		// Refill tokens
		elapsed := time.Since(b.lastTime).Seconds()
		b.tokens += elapsed * float64(rl.rate)
		if b.tokens > float64(rl.burst) {
			b.tokens = float64(rl.burst)
		}
		b.lastTime = time.Now()

		if b.tokens < 1 {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":       429,
					"message":    "too many requests",
					"retry_after": fmt.Sprintf("%.0f", (1-b.tokens)/float64(rl.rate)),
				},
			})
			c.Abort()
			return
		}

		b.tokens--
		rl.mu.Unlock()

		c.Next()
	}
}
