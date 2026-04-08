package service

import (
	"context"
	"fmt"
	"sync"
	"task-manager/internal/queue"
	"task-manager/pkg/logger"
	"time"
)

type NotificationService struct {
	queue   chan queue.Notification
	workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *logger.Logger
}

func NewNotificationService(workers int, logger *logger.Logger) *NotificationService {
	ctx, cancel := context.WithCancel(context.Background())

	ns := &NotificationService{
		queue:   make(chan queue.Notification, 1000),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}

	ns.startWorkers()

	return ns
}

func (ns *NotificationService) startWorkers() {
	for i := 0; i < ns.workers; i++ {
		ns.wg.Add(1)
		go ns.worker(i)
	}

	ns.logger.Info("Notification workers started", "count", ns.workers)
}

func (ns *NotificationService) worker(id int) {
	defer ns.wg.Done()

	for {
		select {
		case notification := <-ns.queue:
			ns.sendNotification(notification)
		case <-ns.ctx.Done():
			ns.logger.Info("Notification worker stopping", "worker_id", id)
			return
		}
	}
}

func (ns *NotificationService) sendNotification(notification queue.Notification) {
	time.Sleep(100 * time.Millisecond)

	ns.logger.Info("Notification sent",
		"type", notification.Type,
		"user_id", notification.UserID,
		"task", notification.TaskTitle,
		"message", notification.Message,
	)
}

func (ns *NotificationService) Send(notification queue.Notification) {
	select {
	case ns.queue <- notification:
	default:
		ns.logger.Warn("Notification queue full, dropping notification",
			"type", notification.Type,
			"user_id", notification.UserID,
		)
	}
}

func (ns *NotificationService) NotifyTaskCreated(userID, taskID, taskTitle string) {
	ns.Send(queue.Notification{
		Type:      queue.NotificationTaskCreated,
		UserID:    userID,
		TaskID:    taskID,
		TaskTitle: taskTitle,
		Message:   fmt.Sprintf("Task '%s' has been created", taskTitle),
		Timestamp: time.Now(),
	})
}

func (ns *NotificationService) NotifyTaskOverdue(userID, taskID, taskTitle string) {
	ns.Send(queue.Notification{
		Type:      queue.NotificationTaskOverdue,
		UserID:    userID,
		TaskID:    taskID,
		TaskTitle: taskTitle,
		Message:   fmt.Sprintf("Task '%s' is overdue!", taskTitle),
		Timestamp: time.Now(),
	})
}

func (ns *NotificationService) Shutdown() {
	ns.logger.Info("Shutting down notification service...")
	ns.cancel()

	done := make(chan struct{})
	go func() {
		ns.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		ns.logger.Info("Notification service stopped gracefully")
	case <-time.After(5 * time.Second):
		ns.logger.Warn("Notification service shutdown timeout")
	}

	close(ns.queue)
}
