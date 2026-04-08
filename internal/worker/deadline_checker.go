package worker

import (
	"context"
	"sync"
	"task-manager/internal/queue"
	"task-manager/internal/repository"
	"task-manager/pkg/logger"
	"time"
)

type DeadlineChecker struct {
	taskRepo *repository.TaskRepository
	notifier queue.Notifier
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *logger.Logger
	mu       sync.RWMutex
	running  bool
}

func NewDeadlineChecker(
	taskRepo *repository.TaskRepository,
	notifier queue.Notifier,
	interval time.Duration,
	logger *logger.Logger,
) *DeadlineChecker {
	ctx, cancel := context.WithCancel(context.Background())

	return &DeadlineChecker{
		taskRepo: taskRepo,
		notifier: notifier,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
	}
}

func (dc *DeadlineChecker) Start() {
	dc.mu.Lock()
	if dc.running {
		dc.mu.Unlock()
		return
	}
	dc.running = true
	dc.mu.Unlock()

	dc.wg.Add(1)
	go dc.run()

	dc.logger.Info("Deadline checker started", "interval", dc.interval)
}

func (dc *DeadlineChecker) run() {
	defer dc.wg.Done()

	ticker := time.NewTicker(dc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dc.checkDeadlines()
		case <-dc.ctx.Done():
			dc.logger.Info("Deadline checker stopping")
			return
		}
	}
}

func (dc *DeadlineChecker) checkDeadlines() {
	dc.logger.Debug("Checking for overdue tasks")

	overdueTasks, err := dc.taskRepo.GetOverdueTasks(context.Background())
	if err != nil {
		dc.logger.Error("Failed to get overdue tasks", "error", err)
		return
	}

	if len(overdueTasks) == 0 {
		dc.logger.Debug("No overdue tasks found")
		return
	}

	dc.logger.Info("Found overdue tasks", "count", len(overdueTasks))

	var wg sync.WaitGroup
	for _, task := range overdueTasks {
		wg.Add(1)
		go func(t repository.OverdueTaskInfo) {
			defer wg.Done()
			dc.notifier.NotifyTaskOverdue(t.UserID, t.ID, t.Title)
		}(task)
	}

	wg.Wait()

	go func() {
		for _, task := range overdueTasks {
			if err := dc.taskRepo.UpdateTaskStatus(context.Background(), task.ID, "overdue"); err != nil {
				dc.logger.Error("Failed to update task status", "task_id", task.ID, "error", err)
			}
		}
	}()

	dc.logger.Info("Overdue notifications processed", "notifications_sent", len(overdueTasks))
}

func (dc *DeadlineChecker) Stop() {
	dc.mu.Lock()
	if !dc.running {
		dc.mu.Unlock()
		return
	}
	dc.running = false
	dc.mu.Unlock()

	dc.cancel()

	done := make(chan struct{})
	go func() {
		dc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		dc.logger.Info("Deadline checker stopped")
	case <-time.After(5 * time.Second):
		dc.logger.Warn("Deadline checker shutdown timeout")
	}
}
