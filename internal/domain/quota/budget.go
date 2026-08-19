package quota

import (
	"errors"
	"sync"
	"time"
)

// Budget tracks a tenant's spendable request allowance.
type Budget struct {
	mu      sync.RWMutex
	limit   int64
	used    int64
	window  time.Duration
	resetAt time.Time
}

func NewBudget(limit int64, window time.Duration, now time.Time) *Budget {
	if limit < 0 {
		limit = 0
	}
	if window <= 0 {
		window = time.Hour
	}
	return &Budget{limit: limit, window: window, resetAt: now.Add(window)}
}

var ErrBudgetExceeded = errors.New("quota budget exceeded")

// Consume reserves one unit if the budget still has capacity.
func (b *Budget) Consume(at time.Time) error {
	if b == nil {
		return ErrBudgetExceeded
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !at.Before(b.resetAt) {
		b.used = 0
		b.resetAt = at.Add(b.window)
	}
	if b.used >= b.limit {
		return ErrBudgetExceeded
	}
	b.used++
	return nil
}

// Remaining returns the units left in the current window.
func (b *Budget) Remaining(at time.Time) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !at.Before(b.resetAt) {
		return b.limit
	}
	remaining := b.limit - b.used
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// ResetAt returns the end of the current budget window.
func (b *Budget) ResetAt() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.resetAt
}
