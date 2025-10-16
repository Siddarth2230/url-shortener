// pkg/cache/lru_test.go
package cache

import (
	"fmt"
	"testing"
)

func TestLRUCache_Basic(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// Get "a" - should exist
	if val, ok := cache.Get("a"); !ok || val != 1 {
		t.Errorf("Expected a=1, got %v", val)
	}

	// Cache is full, add "c" -> should evict "b" (LRU)
	cache.Put("c", 3)

	// "b" should be evicted
	if _, ok := cache.Get("b"); ok {
		t.Error("Expected 'b' to be evicted")
	}

	// "a" and "c" should exist
	if _, ok := cache.Get("a"); !ok {
		t.Error("Expected 'a' to exist")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Error("Expected 'c' to exist")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Put("a", 1)
	cache.Put("a", 10) // Update

	if val, ok := cache.Get("a"); !ok || val != 10 {
		t.Errorf("Expected a=10, got %v", val)
	}
}

func TestLRUCache_Concurrency(t *testing.T) {
	cache := NewLRUCache(100)

	// Spawn 10 goroutines writing
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				cache.Put(key, j)
			}
		}(i)
	}

	// Should not panic (race detector)
}
