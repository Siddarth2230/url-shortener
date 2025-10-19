package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// ErrCacheMiss indicates the key was not found in cache
	ErrCacheMiss = errors.New("cache miss")

	// ErrInvalidValue indicates the cached value couldn't be unmarshaled
	ErrInvalidValue = errors.New("invalid cached value")
)

// RedisCache wraps Redis client for caching
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisCacheConfig holds configuration for Redis cache
type RedisCacheConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
	Prefix   string
}

// NewRedisCache creates a Redis cache with default TTL
func NewRedisCache(client *redis.Client, ttl time.Duration) *RedisCache {
	if ttl == 0 {
		ttl = 5 * time.Minute // default
	}
	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: "urlshort:", // Namespace to avoid key collisions
	}
}

func NewRedisCacheFromConfig(cfg RedisCacheConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "urlshort:"
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

// Get retrieves a value from Redis and unmarshals into v
func (r *RedisCache) Get(ctx context.Context, key string, v interface{}) error {
	// Add prefix to key for namespacing
	fullKey := r.prefix + key

	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil { // Key doesn't exist
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("redis get error: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidValue, err)
	}

	return nil
}

// Set stores a value in Redis with TTL
func (r *RedisCache) Set(ctx context.Context, key string, v interface{}) error {
	return r.SetWithTTL(ctx, key, v, r.ttl)
}

func (r *RedisCache) SetWithTTL(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	fullKey := r.prefix + key

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	if err := r.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete removes a key from Redis
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.prefix + key
	return r.client.Del(ctx, fullKey).Err()
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.prefix + key
	n, err := r.client.Exists(ctx, fullKey).Result()
	return n > 0, err
}
