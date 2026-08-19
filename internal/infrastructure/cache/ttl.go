package cache

import (
	"sync"
	"time"
)

type ttlEntry struct {
	value     any
	expiresAt time.Time
}

// TTL is a thread-safe cache with per-entry expiry.
type TTL struct {
	mu      sync.RWMutex
	entries map[string]ttlEntry
	now     func() time.Time
}

func NewTTL(now func() time.Time) *TTL {
	if now == nil {
		now = time.Now
	}
	return &TTL{entries: map[string]ttlEntry{}, now: now}
}

// Get returns a live value or false when missing or expired.
func (c *TTL) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || c.now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

// Put stores a value until expiresAt.
func (c *TTL) Put(key string, value any, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = ttlEntry{value: value, expiresAt: expiresAt}
}

// Prune removes expired entries and returns how many were dropped.
func (c *TTL) Prune() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for key, entry := range c.entries {
		if c.now().After(entry.expiresAt) {
			delete(c.entries, key)
			removed++
		}
	}
	return removed
}

// Len returns the total number of entries, including expired ones.
func (c *TTL) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
