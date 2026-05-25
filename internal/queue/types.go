package queue

import (
	"time"
)

// NotificationType тип уведомления
type NotificationType string

const (
	NotificationTaskCreated NotificationType = "task_created"
	NotificationTaskUpdated NotificationType = "task_updated"
	NotificationTaskDeleted NotificationType = "task_deleted"
	NotificationTaskOverdue NotificationType = "task_overdue"
)

type Notification struct {
	Type      NotificationType
	UserID    string
	TaskID    string
	TaskTitle string
	Message   string
	Timestamp time.Time
}

type Job struct {
	ID        string
	TaskID    string
	UserID    string
	Type      string
	Data      interface{}
	CreatedAt time.Time
}

type Result struct {
	JobID       string
	Success     bool
	Error       error
	ProcessedAt time.Time
}

type WorkerMetrics struct {
	TotalJobs       int64
	CompletedJobs   int64
	FailedJobs      int64
	AverageDuration time.Duration
}

type Notifier interface {
	Send(notification Notification)
	NotifyTaskCreated(userID, taskID, taskTitle string)
	NotifyTaskOverdue(userID, taskID, taskTitle string)
	Shutdown()
}
