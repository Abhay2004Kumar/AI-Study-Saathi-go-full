// Package worker implements a bounded background worker pool for processing jobs.
package worker

import (
	"context"
	"log/slog"
	"sync"
)

// JobProcessor defines the task a worker executes.
type JobProcessor func(ctx context.Context, jobID string) error

// Pool manages a bounded pool of worker goroutines.
type Pool struct {
	queue     chan string
	processor JobProcessor
	workers   int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPool creates a new bounded worker pool.
func NewPool(workers int, queueSize int, processor JobProcessor) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		queue:     make(chan string, queueSize),
		processor: processor,
		workers:   workers,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start launches the worker goroutines.
func (p *Pool) Start() {
	for i := 1; i <= p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	slog.Info("Worker pool started", "workers", p.workers)
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case jobID, ok := <-p.queue:
			if !ok {
				return
			}
			slog.Info("Worker processing job", "workerId", id, "jobId", jobID)
			if err := p.processor(p.ctx, jobID); err != nil {
				slog.Error("Worker job failed", "workerId", id, "jobId", jobID, "error", err)
			} else {
				slog.Info("Worker job completed", "workerId", id, "jobId", jobID)
			}
		}
	}
}

// Enqueue adds a job ID to the queue. If queue is full, it drops or blocks based on design.
func (p *Pool) Enqueue(jobID string) {
	select {
	case p.queue <- jobID:
		slog.Info("Job enqueued", "jobId", jobID)
	default:
		slog.Warn("Worker queue is full, running job in separate goroutine", "jobId", jobID)
		go func() {
			_ = p.processor(p.ctx, jobID)
		}()
	}
}

// Stop gracefully shuts down the worker pool, waiting for in-flight jobs to finish.
func (p *Pool) Stop() {
	p.cancel()
	close(p.queue)
	p.wg.Wait()
	slog.Info("Worker pool stopped")
}

