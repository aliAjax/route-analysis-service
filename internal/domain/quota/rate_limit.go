package quota

import (
	"errors"
	"sync"
	"time"
)

// RateLimiter is a sliding-window request limiter keyed by tenant.
type RateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string][]time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{window: window, limit: limit, entries: nil}
}

var ErrRateLimited = errors.New("rate limit exceeded")

// Allow records one request and reports whether it is within the limit.
func (r *RateLimiter) Allow(key string, at time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := at.Add(-r.window)
	kept := r.entries[key][:0]
	for _, ts := range r.entries[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= r.limit {
		r.entries[key] = kept
		return false
	}
	r.entries[key] = append(kept, at)
	return true
}

// Count returns the number of requests in the current window.
func (r *RateLimiter) Count(key string, at time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := at.Add(-r.window)
	count := 0
	for _, ts := range r.entries[key] {
		if ts.After(cutoff) {
			count++
		}
	}
	return count
}
