package idgen

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type CounterGenerator struct {
	redis *redis.Client
}

func NewCounterGenerator(redisClient *redis.Client) *CounterGenerator {
	return &CounterGenerator{redis: redisClient}
}

// Generate returns next ID using Redis INCR (atomic counter)
func (g *CounterGenerator) Generate(ctx context.Context) (string, error) {
	// Use Redis INCR to atomically get the next unique integer
	val, err := g.redis.Incr(ctx, "url_counter").Result()
	if err != nil {
		return "", fmt.Errorf("failed to increment counter: %w", err)
	}

	// Convert numeric ID to Base62 short code
	shortCode := Encode(uint64(val))

	return shortCode, nil
}
