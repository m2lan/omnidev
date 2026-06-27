// Package cache provides Redis client management.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

// Redis wraps the Redis client with convenience methods.
type Redis struct {
	Client *redis.Client
}

// NewRedis creates a new Redis client.
func NewRedis(cfg config.RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Log.Info("Redis connected",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	return &Redis{Client: client}, nil
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	return r.Client.Close()
}

// Ping checks the Redis connection.
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Set stores a value with expiration.
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.Client.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value and unmarshals it into dest.
func (r *Redis) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key: %w", err)
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists.
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetNX sets a value only if the key does not exist.
func (r *Redis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.Client.SetNX(ctx, key, data, expiration).Result()
}

// Incr atomically increments a key.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// IncrBy atomically increments a key by a value.
func (r *Redis) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.Client.IncrBy(ctx, key, value).Result()
}

// Expire sets expiration on a key.
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// HSet sets a field in a hash.
func (r *Redis) HSet(ctx context.Context, key string, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.Client.HSet(ctx, key, field, data).Err()
}

// HGet gets a field from a hash.
func (r *Redis) HGet(ctx context.Context, key string, field string, dest interface{}) error {
	data, err := r.Client.HGet(ctx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("field not found: %s", field)
		}
		return fmt.Errorf("failed to get hash field: %w", err)
	}
	return json.Unmarshal(data, dest)
}

// HDel deletes a field from a hash.
func (r *Redis) HDel(ctx context.Context, key string, fields ...string) error {
	return r.Client.HDel(ctx, key, fields...).Err()
}

// Publish publishes a message to a channel.
func (r *Redis) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return r.Client.Publish(ctx, channel, data).Err()
}

// Subscribe subscribes to a channel.
func (r *Redis) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.Client.Subscribe(ctx, channels...)
}
