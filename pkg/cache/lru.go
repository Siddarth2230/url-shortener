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

	// TODO: Implement this
	// Steps:
	// 1. If key exists, update value and move to front
	// 2. If key is new:
	//    a. Check if cache is full (len(cache) >= capacity)
	//    b. If full, evict tail (removeNode + delete from map)
	//    c. Create new node and add to front
	//    d. Add to map
}

// moveToFront moves a node to the head (most recent)
func (c *LRUCache) moveToFront(node *Node) {
	// TODO: Implement this
	// Steps:
	// 1. Remove node from current position (removeNode)
	// 2. Add node to front (addToFront)
}

// removeNode removes a node from the list (doesn't delete from map)
func (c *LRUCache) removeNode(node *Node) {
	// TODO: Implement this
	// Hint: Update prev and next pointers
	// node.Prev.Next = ???
	// node.Next.Prev = ???
}

// addToFront adds a node right after the dummy head
func (c *LRUCache) addToFront(node *Node) {
	// TODO: Implement this
	// Hint: Insert between head and head.Next
	// node.Next = ???
	// node.Prev = ???
	// head.Next.Prev = ???
	// head.Next = ???
}

// evictTail removes the least recently used item
func (c *LRUCache) evictTail() {
	// TODO: Implement this
	// Steps:
	// 1. Get tail.Prev (actual last node, tail is dummy)
	// 2. Remove from list
	// 3. Delete from map
}

// Len returns current number of cached items
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Clear empties the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*Node, c.capacity)
	c.head.Next = c.tail
	c.tail.Prev = c.head
}
