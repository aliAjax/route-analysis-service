package worker

import (
	"context"
	"sync"
	"time"
)

// Dispatcher fans a slice of inputs out to a bounded number of workers.
type Dispatcher struct {
	workers int
}

func NewDispatcher(workers int) *Dispatcher {
	if workers < 1 {
		workers = 1
	}
	return &Dispatcher{workers: workers}
}

// Run invokes fn for every input, bounded by the configured worker count, and
// returns the number of successful invocations. Cancellation stops scheduling.
func (d *Dispatcher) Run(ctx context.Context, inputs []string, fn func(context.Context, string) error) (int, error) {
	inputsCh := make(chan string)
	var wg sync.WaitGroup
	var success int64
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, d.workers)
	workerFn := func() {
		defer wg.Done()
		for input := range inputsCh {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			err := fn(ctx, input)
			<-sem
			mu.Lock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
			} else {
				success++
			}
			mu.Unlock()
			if ctx.Err() != nil {
				return
			}
		}
	}
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go workerFn()
	}
loop:
	for _, input := range inputs {
		select {
		case inputsCh <- input:
		case <-ctx.Done():
			break loop
		}
	}
	close(inputsCh)
	wg.Wait()
	return int(success), firstErr
}

// SleepCtx sleeps for the duration or until ctx is cancelled.
func SleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
