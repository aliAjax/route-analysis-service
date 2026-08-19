package cache

import (
	"container/list"
	"sync"
)

// LRU is a thread-safe least-recently-used cache.
type LRU struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type lruEntry struct {
	key   string
	value any
}

func NewLRU(capacity int) *LRU {
	if capacity < 1 {
		capacity = 64
	}
	return &LRU{capacity: capacity, items: map[string]*list.Element{}, order: list.New()}
}

// Get returns a cached value and reports whether it was present.
func (c *LRU) Get(key string) (any, bool) {
	element, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(*lruEntry).value, true
}

// Put stores a value, evicting the least recently used entry when full.
func (c *LRU) Put(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		element.Value.(*lruEntry).value = value
		c.order.MoveToFront(element)
		return
	}
	element := c.order.PushFront(&lruEntry{key: key, value: value})
	c.items[key] = element
	for c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*lruEntry).key)
	}
}

// Len returns the number of cached entries.
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
