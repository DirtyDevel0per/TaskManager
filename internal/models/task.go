package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusInProgress TaskStatus = "in_progress"
)

type Task struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description" validate:"required"`
	DueDate     *time.Time `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string     `json:"title" validate:"omitempty"`
	Description *string     `json:"description" validate:"omitempty"`
	Status      *TaskStatus `json:"status" validate:"omitempty,oneof=pending in_progress completed"`
	DueDate     *time.Time  `json:"due_date"`
}
