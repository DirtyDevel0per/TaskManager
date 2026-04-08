package worker

import (
	"context"
	"sync"
	"task-manager/internal/models"
	"task-manager/internal/repository"
	"task-manager/pkg/logger"
	"time"
)

type BatchProcessor struct {
	taskRepo    *repository.TaskRepository
	batchSize   int
	buffer      chan *models.Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *logger.Logger
	flushTicker *time.Ticker
}

func NewBatchProcessor(
	taskRepo *repository.TaskRepository,
	batchSize int,
	bufferSize int,
	flushInterval time.Duration,
	logger *logger.Logger,
) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		taskRepo:    taskRepo,
		batchSize:   batchSize,
		buffer:      make(chan *models.Task, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		flushTicker: time.NewTicker(flushInterval),
	}

	bp.start()

	return bp
}

func (bp *BatchProcessor) start() {
	bp.wg.Add(1)
	go bp.process()

	bp.logger.Info("Batch processor started",
		"batch_size", bp.batchSize,
		"buffer_size", cap(bp.buffer),
	)
}

func (bp *BatchProcessor) process() {
	defer bp.wg.Done()

	batch := make([]*models.Task, 0, bp.batchSize)

	for {
		select {
		case task := <-bp.buffer:
			batch = append(batch, task)

			if len(batch) >= bp.batchSize {
				bp.flush(batch)
				batch = make([]*models.Task, 0, bp.batchSize)
			}

		case <-bp.flushTicker.C:
			if len(batch) > 0 {
				bp.flush(batch)
				batch = make([]*models.Task, 0, bp.batchSize)
			}

		case <-bp.ctx.Done():
			if len(batch) > 0 {
				bp.flush(batch)
			}
			bp.logger.Info("Batch processor stopping")
			return
		}
	}
}

func (bp *BatchProcessor) flush(batch []*models.Task) {
	var wg sync.WaitGroup
	errors := make(chan error, len(batch))

	for _, task := range batch {
		wg.Add(1)
		go func(t *models.Task) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := bp.taskRepo.Create(ctx, t); err != nil {
				errors <- err
			}
		}(task)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	var errCount int
	for range errors {
		errCount++
	}

	bp.logger.Info("Batch inserted",
		"total", len(batch),
		"failed", errCount,
		"success", len(batch)-errCount,
	)
}

func (bp *BatchProcessor) AddTask(task *models.Task) {
	select {
	case bp.buffer <- task:
	default:
		bp.logger.Warn("Batch buffer full, task rejected", "task_id", task.ID)
	}
}

func (bp *BatchProcessor) Shutdown() {
	bp.logger.Info("Shutting down batch processor...")
	bp.cancel()
	bp.flushTicker.Stop()

	done := make(chan struct{})
	go func() {
		bp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		bp.logger.Info("Batch processor stopped")
	case <-time.After(5 * time.Second):
		bp.logger.Warn("Batch processor shutdown timeout")
	}

	close(bp.buffer)
}
