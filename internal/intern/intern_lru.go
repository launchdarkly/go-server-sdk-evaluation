//go:build go1.23 && launchdarkly_intern_json

package intern

import (
	"strings"
	"sync"
	"sync/atomic"
	"unique"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

// LRU cache-backed interning that retains handles for frequently used strings.
// String() uses an LRU cache to store handles so canonical bytes survive
// across GCs and are reused by future lookups. Cache size is configurable
// via LAUNCHDARKLY_INTERN_CACHE_SIZE environment variable.

// LRU cache node for doubly-linked list
type lruNode struct {
	key    string
	handle unique.Handle[string]
	prev   *lruNode
	next   *lruNode
}

// LRU cache implementation for bounded string interning
type lruCache struct {
	mu       sync.RWMutex
	capacity int
	cache    map[string]*lruNode
	head     *lruNode // dummy head node
	tail     *lruNode // dummy tail node
	// Statistics
	hits    uint64
	misses  uint64
	evicted uint64
}

func newLRUCache(capacity int) *lruCache {
	if capacity <= 0 {
		capacity = 8192 // default capacity
	}

	lru := &lruCache{
		capacity: capacity,
		cache:    make(map[string]*lruNode, capacity),
	}

	// Create dummy head and tail nodes
	lru.head = &lruNode{}
	lru.tail = &lruNode{}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head

	return lru
}

func (lru *lruCache) addToHead(node *lruNode) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

func (lru *lruCache) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (lru *lruCache) moveToHead(node *lruNode) {
	lru.removeNode(node)
	lru.addToHead(node)
}

func (lru *lruCache) removeTail() *lruNode {
	lastNode := lru.tail.prev
	if lastNode == lru.head {
		// Cache is empty, nothing to remove
		return nil
	}
	lru.removeNode(lastNode)
	return lastNode
}

func (lru *lruCache) get(key string) (unique.Handle[string], bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if node, exists := lru.cache[key]; exists {
		// Move to head (most recently used)
		lru.moveToHead(node)
		lru.hits++
		return node.handle, true
	}
	lru.misses++
	return unique.Handle[string]{}, false
}

func (lru *lruCache) put(key string, handle unique.Handle[string]) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if node, exists := lru.cache[key]; exists {
		// Update existing node
		node.handle = handle
		lru.moveToHead(node)
		return
	}

	// Create new node
	newNode := &lruNode{
		key:    key,
		handle: handle,
	}

	if len(lru.cache) >= lru.capacity {
		// Remove least recently used
		tail := lru.removeTail()
		if tail != nil {
			delete(lru.cache, tail.key)
			lru.evicted++
		}
	}

	lru.cache[key] = newNode
	lru.addToHead(newNode)
}

func (lru *lruCache) stats() (hits, misses, evicted uint64, size, capacity int) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return lru.hits, lru.misses, lru.evicted, len(lru.cache), lru.capacity
}

func getRetainCacheSize() int {
	return 8192 // default: 8K entries
}

var (
	cache          = newLRUCache(getRetainCacheSize())
	internDisabled atomic.Bool
	cacheMutex     sync.RWMutex
)

func getCache() *lruCache {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	return cache
}

// SetCacheSize allows runtime control of cache size and interning behavior.
// If size <= 0, interning is disabled entirely.
// If size > 0, creates a new cache with the specified capacity.
func SetCacheSize(size int) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if size <= 0 {
		internDisabled.Store(true)
		cache = nil
		return
	}

	internDisabled.Store(false)
	cache = newLRUCache(size)
}

// String returns a canonical form of s and retains a handle so the
// canonical bytes persist across GCs.
func String(s string) string {
	if internDisabled.Load() {
		return s
	}

	if s == "" {
		return ""
	}

	cache := getCache()

	// Check cache first
	if handle, exists := cache.get(s); exists {
		return handle.Value()
	}

	// Create new handle and cache it
	h := unique.Make(strings.Clone(s))
	v := h.Value()
	cache.put(v, h)

	return v
}

// Value canonicalizes only string ldvalue.Value; others are returned unchanged.
// String values retain their handle (same as String).
func Value(v ldvalue.Value) ldvalue.Value {
	switch v.Type() {
	case ldvalue.StringType:
		return ldvalue.String(String(v.StringValue()))
	default:
		return v
	}
}

// StringSlice applies String (retained) to all elements.
func StringSlice(ss []string) []string {
	for i, s := range ss {
		ss[i] = String(s)
	}
	return ss
}

// ValueSlice applies Value (retained for strings) to all elements.
func ValueSlice(vs []ldvalue.Value) []ldvalue.Value {
	for i, v := range vs {
		vs[i] = Value(v)
	}
	return vs
}

// CacheStats returns cache statistics for monitoring and debugging.
// Returns hits, misses, evicted count, current size, and capacity.
func CacheStats() (hits, misses, evicted uint64, size, capacity int) {
	return getCache().stats()
}
