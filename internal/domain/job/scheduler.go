package job

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// WorkerFunc processes one job and returns a result map or an error.
type WorkerFunc func(context.Context, model.AnalysisJob) (map[string]any, error)

// Scheduler drains a queue using a fixed-size worker pool.
type Scheduler struct {
	queue   *Queue
	workers int
	process WorkerFunc
	mu      sync.Mutex
	active  int
}

func NewScheduler(queue *Queue, workers int, process WorkerFunc) *Scheduler {
	if workers < 1 {
		workers = 1
	}
	return &Scheduler{queue: queue, workers: workers, process: process}
}

var ErrShuttingDown = errors.New("scheduler is shutting down")

// Run blocks until ctx is cancelled, continuously dispatching queued jobs to a
// bounded set of goroutines. It returns when all in-flight jobs finish.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.workers)
	for {
		env, ok := s.queue.Pop()
		if !ok {
			select {
			case <-ctx.Done():
				wg.Wait()
				return
			case <-time.After(20 * time.Millisecond):
				continue
			}
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		wg.Add(1)
		go func(env Envelope) {
			defer wg.Done()
			defer func() { <-sem }()
			s.mu.Lock()
			s.active++
			s.mu.Unlock()
			_, _ = s.process(ctx, env.Job)
			s.mu.Lock()
			s.active--
			s.mu.Unlock()
		}(env)
	}
}

// Active returns the number of jobs currently being processed.
func (s *Scheduler) Active() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}
