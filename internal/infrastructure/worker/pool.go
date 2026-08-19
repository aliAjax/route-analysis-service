package worker

import (
	"context"
	"errors"
	"sync"
)

// Task is a unit of work that can fail.
type Task func(context.Context) error

// Pool executes tasks with a fixed number of goroutines and propagates
// cancellation through the context.
type Pool struct {
	workers int
	queue   chan Task
	wg      sync.WaitGroup
	once    sync.Once
	closed  chan struct{}
}

func NewPool(workers int, queueSize int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 1 {
		queueSize = workers
	}
	p := &Pool{workers: workers, queue: make(chan Task, queueSize), closed: make(chan struct{})}
	p.start()
	return p
}

func (p *Pool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.queue {
		_ = task(context.Background())
	}
}

// Submit enqueues a task. It returns an error after the pool is closed.
func (p *Pool) Submit(task Task) error {
	select {
	case <-p.closed:
		return errors.New("worker pool is closed")
	default:
	}
	select {
	case p.queue <- task:
		return nil
	case <-p.closed:
		return errors.New("worker pool is closed")
	}
}

// Close stops accepting tasks and waits for queued tasks to finish.
func (p *Pool) Close() {
	p.once.Do(func() {
		close(p.closed)
		close(p.queue)
		p.wg.Wait()
	})
}
