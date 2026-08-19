package job

import (
	"errors"
	"sort"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Priority describes queue ordering for analysis jobs.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

// Envelope couples a job with scheduling metadata used by the dispatcher.
type Envelope struct {
	Job        model.AnalysisJob
	Priority   Priority
	EnqueuedAt time.Time
}

// Queue is a bounded, priority-ordered work queue with optional deduplication.
type Queue struct {
	items []Envelope
	seen  map[model.JobID]bool
	limit int
}

func NewQueue(limit int) *Queue {
	if limit < 1 {
		limit = 64
	}
	return &Queue{items: make([]Envelope, 0, limit), seen: map[model.JobID]bool{}, limit: limit}
}

var ErrQueueFull = errors.New("job queue is full")

// Push inserts a job envelope unless the same job id is already queued.
func (q *Queue) Push(env Envelope) error {
	if q.seen[env.Job.ID] {
		return errors.New("job already queued")
	}
	if len(q.items) >= q.limit {
		return ErrQueueFull
	}
	q.items = append(q.items, env)
	q.seen[env.Job.ID] = true
	sort.SliceStable(q.items, func(i, j int) bool {
		if q.items[i].Priority == q.items[j].Priority {
			return q.items[i].EnqueuedAt.Before(q.items[j].EnqueuedAt)
		}
		return q.items[i].Priority > q.items[j].Priority
	})
	return nil
}

// Pop removes and returns the highest-priority envelope.
func (q *Queue) Pop() (Envelope, bool) {
	if len(q.items) == 0 {
		return Envelope{}, false
	}
	env := q.items[0]
	q.items = q.items[1:]
	delete(q.seen, env.Job.ID)
	return env, true
}

// Peek returns the highest-priority envelope without removing it.
func (q *Queue) Peek() (Envelope, bool) {
	if len(q.items) == 0 {
		return Envelope{}, false
	}
	return q.items[0], true
}

// Len returns the number of queued jobs.
func (q *Queue) Len() int { return len(q.items) }

// Contains reports whether the job id is already queued.
func (q *Queue) Contains(id model.JobID) bool { return q.seen[id] }

// Snapshot returns a defensive copy of the queued envelopes in priority order.
func (q *Queue) Snapshot() []Envelope {
	return append([]Envelope(nil), q.items...)
}
