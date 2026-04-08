package worker

import (
	"context"
	"fmt"
	"sync"
	"task-manager/internal/queue"
	"task-manager/pkg/logger"
	"time"
)

type TaskWorkerPool struct {
	jobQueue    chan queue.Job
	resultQueue chan queue.Result
	workers     int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *logger.Logger
	metrics     struct {
		mu              sync.RWMutex
		totalJobs       int64
		completedJobs   int64
		failedJobs      int64
		averageDuration time.Duration
	}
}

func NewTaskWorkerPool(workers int, logger *logger.Logger) *TaskWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &TaskWorkerPool{
		jobQueue:    make(chan queue.Job, workers*10),
		resultQueue: make(chan queue.Result, workers*10),
		workers:     workers,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
	}

	pool.start()

	return pool
}

func (p *TaskWorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	go p.processResults()

	p.logger.Info("Task worker pool started", "workers", p.workers)
}

func (p *TaskWorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case job := <-p.jobQueue:
			startTime := time.Now()
			result := p.processJob(job)
			result.ProcessedAt = time.Now()

			p.updateMetrics(startTime, result.Success)

			select {
			case p.resultQueue <- result:
			default:
				p.logger.Warn("Result queue full, dropping result", "job_id", job.ID)
			}

		case <-p.ctx.Done():
			p.logger.Info("Worker stopping", "worker_id", id)
			return
		}
	}
}

func (p *TaskWorkerPool) processJob(job queue.Job) queue.Result {
	time.Sleep(500 * time.Millisecond)

	switch job.Type {
	case "export":
		return p.exportTasks(job)
	case "analyze":
		return p.analyzeTasks(job)
	case "batch_update":
		return p.batchUpdate(job)
	default:
		return queue.Result{
			JobID:   job.ID,
			Success: false,
			Error:   fmt.Errorf("unknown job type: %s", job.Type),
		}
	}
}

func (p *TaskWorkerPool) exportTasks(job queue.Job) queue.Result {
	p.logger.Info("Exporting tasks", "user_id", job.UserID, "job_id", job.ID)
	return queue.Result{JobID: job.ID, Success: true, Error: nil}
}

func (p *TaskWorkerPool) analyzeTasks(job queue.Job) queue.Result {
	p.logger.Info("Analyzing tasks", "user_id", job.UserID, "job_id", job.ID)
	return queue.Result{JobID: job.ID, Success: true, Error: nil}
}

func (p *TaskWorkerPool) batchUpdate(job queue.Job) queue.Result {
	p.logger.Info("Batch updating tasks", "user_id", job.UserID, "job_id", job.ID)
	return queue.Result{JobID: job.ID, Success: true, Error: nil}
}

func (p *TaskWorkerPool) updateMetrics(startTime time.Time, success bool) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.totalJobs++
	if success {
		p.metrics.completedJobs++
	} else {
		p.metrics.failedJobs++
	}

	duration := time.Since(startTime)
	if p.metrics.averageDuration == 0 {
		p.metrics.averageDuration = duration
	} else {
		p.metrics.averageDuration = (p.metrics.averageDuration + duration) / 2
	}
}

func (p *TaskWorkerPool) processResults() {
	for result := range p.resultQueue {
		if !result.Success {
			p.logger.Error("Job failed", "job_id", result.JobID, "error", result.Error)
		}
	}
}

func (p *TaskWorkerPool) Submit(job queue.Job) {
	select {
	case p.jobQueue <- job:
		p.logger.Debug("Job submitted", "job_id", job.ID, "type", job.Type)
	default:
		p.logger.Warn("Job queue full, job rejected", "job_id", job.ID)
	}
}

func (p *TaskWorkerPool) GetMetrics() queue.WorkerMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return queue.WorkerMetrics{
		TotalJobs:       p.metrics.totalJobs,
		CompletedJobs:   p.metrics.completedJobs,
		FailedJobs:      p.metrics.failedJobs,
		AverageDuration: p.metrics.averageDuration,
	}
}

func (p *TaskWorkerPool) Shutdown() {
	p.logger.Info("Shutting down worker pool...")
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(p.jobQueue)
		close(p.resultQueue)
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
	case <-time.After(10 * time.Second):
		p.logger.Warn("Worker pool shutdown timeout")
	}
}
