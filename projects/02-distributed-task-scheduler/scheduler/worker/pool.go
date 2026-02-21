// Package worker provides the task execution worker pool.
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Polqt/scheduler/scheduler/store"
)

// HandlerFunc is the function signature for task handlers.
type HandlerFunc func(ctx context.Context, task *store.Task) error

// Config controls the worker pool behavior.
type Config struct {
	Concurrency       int           // number of concurrent workers
	PollInterval      time.Duration // how often to check for new tasks
	ShutdownTimeout   time.Duration // max time to wait for in-flight tasks
	VisibilityTimeout time.Duration // re-enqueue tasks claimed but not completed
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig() Config {
	return Config{
		Concurrency:       10,
		PollInterval:      500 * time.Millisecond,
		ShutdownTimeout:   30 * time.Second,
		VisibilityTimeout: 5 * time.Minute,
	}
}

// Pool is a concurrent worker pool that polls the store for tasks.
type Pool struct {
	config   Config
	store    store.Store
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewPool creates a new worker pool.
func NewPool(s store.Store, cfg Config) *Pool {
	return &Pool{
		config:   cfg,
		store:    s,
		handlers: make(map[string]HandlerFunc),
	}
}

// Register registers a handler for the given task type.
func (p *Pool) Register(taskType string, fn HandlerFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[taskType] = fn
}

// Start launches `config.Concurrency` worker goroutines.
// Returns immediately; workers run until ctx is cancelled.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.config.Concurrency; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.runWorker(ctx, fmt.Sprintf("worker-%d", id))
		}(i)
	}
}

// Drain waits for all in-flight tasks to complete (up to ShutdownTimeout).
func (p *Pool) Drain() {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(p.config.ShutdownTimeout):
		fmt.Printf("shutdown timeout: some tasks may not have completed\n")
	}
}

// runWorker is the main loop for a single worker goroutine.
func (p *Pool) runWorker(ctx context.Context, workerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		task, err := p.store.ClaimNext(workerID)
		if err != nil {
			// TODO: log error, back off briefly
			time.Sleep(p.config.PollInterval)
			continue
		}
		if task == nil {
			// Nothing ready; wait before polling again
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.config.PollInterval):
			}
			continue
		}

		p.executeTask(ctx, task)
	}
}

// executeTask runs a single task with the registered handler.
func (p *Pool) executeTask(ctx context.Context, task *store.Task) {
	p.mu.RLock()
	handler, ok := p.handlers[task.Type]
	p.mu.RUnlock()

	if !ok {
		// TODO: log "no handler for task type" and mark task as failed
		p.store.MarkFailed(task.ID, fmt.Sprintf("no handler registered for type %q", task.Type))
		return
	}

	// TODO:
	// 1. Create a context with a timeout (derive from config or task payload)
	// 2. Call handler(ctx, task)
	// 3. On nil error: call store.MarkDone(task.ID)
	//    If task has a schedule: parse cron + call store.Reschedule with next time
	// 4. On error: call store.MarkFailed(task.ID, err.Error())

	err := handler(ctx, task)
	if err != nil {
		p.store.MarkFailed(task.ID, err.Error())
		return
	}
	if task.Schedule != "" {
		// TODO: parse task.Schedule as cron, compute next run time
		// store.Reschedule(task.ID, nextRunAt)
	} else {
		p.store.MarkDone(task.ID)
	}
}
