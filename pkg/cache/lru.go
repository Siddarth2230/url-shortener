package cache

import (
	"sync"
)

// Node represents a doubly linked list node
type Node struct {
	Key   string
	Value interface{}
	Prev  *Node
	Next  *Node
}

// LRUCache is a thread-safe LRU cache
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	cache    map[string]*Node
	head     *Node // most recently used
	tail     *Node // least recently used
}

// NewLRUCache creates an LRU cache with given capacity
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 1000 // default
	}

	c := &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node, capacity),
	}

	// Initialize dummy head and tail
	c.head = &Node{}
	c.tail = &Node{}
	c.head.Next = c.tail
	c.tail.Prev = c.head

	return c
}

// Get retrieves value and marks as recently used
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(node)
	return node.Value, true
}

// Put adds or updates a key-value pair
func (c *LRUCache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, update value and move to front
	if node, exists := c.cache[key]; exists {
		node.Value = value
		c.moveToFront(node)
		return
	}
	// If key is new:
	if len(c.cache) >= c.capacity {
		c.evictTail()
	}
	// Create new node and add to front
	newNode := &Node{
		Key:   key,
		Value: value,
	}
	c.addToFront(newNode)

	// Add to map
	c.cache[key] = newNode
}

// moveToFront moves a node to the head (most recent)
func (c *LRUCache) moveToFront(node *Node) {
	// Remove node from current position
	c.removeNode(node)
	// Add node to front (addToFront)
	c.addToFront(node)
}

// removeNode removes a node from the list (doesn't delete from map)
func (c *LRUCache) removeNode(node *Node) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
}

// addToFront adds a node right after the dummy head
func (c *LRUCache) addToFront(node *Node) {
	HeadNext := c.head.Next

	node.Next = HeadNext
	node.Prev = c.head

	c.head.Next = node
	HeadNext.Prev = node
}

// evictTail removes the least recently used item
func (c *LRUCache) evictTail() {
	lru := c.tail.Prev

	if lru == c.head {
		return
	}
	// Delete from list
	c.removeNode(lru)

	// Delete from map
	delete(c.cache, lru.Key)
}

// Clear empties the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*Node, c.capacity)
	c.head.Next = c.tail
	c.tail.Prev = c.head
}

// Deletes a specific node
func (c *LRUCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.cache[key]
	if !exists {
		return false
	}

	c.removeNode(node)
	delete(c.cache, key)
	return true
}

// Peek retrieves value WITHOUT marking as recently used. Useful for checking cache without affecting eviction order
func (c *LRUCache) Peek(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	return node.Value, true
}

func (c *LRUCache) Len() int {
	return len(c.cache)
}
