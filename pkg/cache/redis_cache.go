package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache wraps Redis client for caching
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache creates a Redis cache with default TTL
func NewRedisCache(client *redis.Client, ttl time.Duration) *RedisCache {
	if ttl == 0 {
		ttl = 5 * time.Minute // default
	}
	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

// Get retrieves a value from Redis and unmarshals into v
func (r *RedisCache) Get(ctx context.Context, key string, v interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// Set stores a value in Redis with TTL
func (r *RedisCache) Set(ctx context.Context, key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// Delete removes a key from Redis
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

var ErrCacheMiss = errors.New("cache miss")
