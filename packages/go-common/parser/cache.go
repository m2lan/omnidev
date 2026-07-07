// Package parser provides document parsing capabilities using Apache Tika.
package parser

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

const (
	// cacheKeyPrefix is the prefix for parser cache keys.
	cacheKeyPrefix = "parser:doc:"
	// defaultCacheTTL is the default TTL for cached parse results.
	defaultCacheTTL = 7 * 24 * time.Hour // 7 days
)

// CachedParser wraps a Parser with Redis caching.
type CachedParser struct {
	inner Parser
	redis *redis.Client
	ttl   time.Duration
}

// NewCachedParser creates a new cached parser.
func NewCachedParser(inner Parser, redisClient *redis.Client, ttl time.Duration) *CachedParser {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}

	return &CachedParser{
		inner: inner,
		redis: redisClient,
		ttl:   ttl,
	}
}

// Parse parses a document with caching support.
func (p *CachedParser) Parse(ctx context.Context, filename string, reader io.Reader) (*ParseResult, error) {
	// Read all content to compute hash and pass to parser
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Compute content hash
	hash := sha256.Sum256(content)
	cacheKey := cacheKeyPrefix + hex.EncodeToString(hash[:])

	// Try to get from cache
	if p.redis != nil {
		cached, err := p.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var result ParseResult
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				logger.Log.Debug("Parser cache hit", zap.String("key", cacheKey[:16]+"..."))
				return &result, nil
			}
		}
	}

	// Cache miss, parse the document
	result, err := p.inner.Parse(ctx, filename, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	// Store in cache
	if p.redis != nil {
		data, err := json.Marshal(result)
		if err == nil {
			if err := p.redis.Set(ctx, cacheKey, string(data), p.ttl).Err(); err != nil {
				logger.Log.Warn("Failed to cache parse result", zap.Error(err))
			} else {
				logger.Log.Debug("Parser cache stored", zap.String("key", cacheKey[:16]+"..."))
			}
		}
	}

	return result, nil
}

// SupportedFormats delegates to the inner parser.
func (p *CachedParser) SupportedFormats() []string {
	return p.inner.SupportedFormats()
}
