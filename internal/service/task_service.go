package service

import (
	"context"
	"fmt"
	"task-manager/internal/models"
	"task-manager/internal/queue"
	"task-manager/internal/repository"
	"task-manager/internal/worker"
	"task-manager/pkg/logger"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo     *repository.TaskRepository
	notification *NotificationService
	workerPool   *worker.TaskWorkerPool
	batchProc    *worker.BatchProcessor
	logger       *logger.Logger
}

func NewTaskService(
	taskRepo *repository.TaskRepository,
	notification *NotificationService,
	workerPool *worker.TaskWorkerPool,
	batchProc *worker.BatchProcessor,
	logger *logger.Logger,
) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		notification: notification,
		workerPool:   workerPool,
		batchProc:    batchProc,
		logger:       logger,
	}
}

func (s *TaskService) Create(ctx context.Context, userID uuid.UUID, req *models.CreateTaskRequest) (*models.Task, error) {
	task := &models.Task{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatusPending,
		DueDate:     req.DueDate,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	go s.notification.NotifyTaskCreated(userID.String(), task.ID.String(), task.Title)

	job := queue.Job{
		ID:        uuid.New().String(),
		TaskID:    task.ID.String(),
		UserID:    userID.String(),
		Type:      "analyze",
		Data:      task,
		CreatedAt: time.Now(),
	}
	s.workerPool.Submit(job)

	return task, nil
}

func (s *TaskService) GetByID(ctx context.Context, userID, taskID uuid.UUID) (*models.Task, error) {
	return s.taskRepo.GetByID(ctx, taskID, userID)
}

func (s *TaskService) List(ctx context.Context, userID uuid.UUID, status *models.TaskStatus, page, pageSize int) ([]*models.Task, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.taskRepo.List(ctx, userID, status, pageSize, offset)
}

func (s *TaskService) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req *models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, userID, taskID uuid.UUID) error {
	return s.taskRepo.Delete(ctx, taskID, userID)
}

func (s *TaskService) BatchCreate(userID uuid.UUID, tasks []models.CreateTaskRequest) (int, error) {
	created := 0

	for _, req := range tasks {
		task := &models.Task{
			UserID:      userID,
			Title:       req.Title,
			Description: req.Description,
			Status:      models.TaskStatusPending,
			DueDate:     req.DueDate,
		}
		s.batchProc.AddTask(task)
		created++
	}

	return created, nil
}

func (s *TaskService) ExportTasks(userID uuid.UUID, format string) (string, error) {
	jobID := uuid.New().String()

	job := queue.Job{
		ID:     jobID,
		UserID: userID.String(),
		Type:   "export",
		Data: map[string]interface{}{
			"format":  format,
			"user_id": userID,
		},
		CreatedAt: time.Now(),
	}

	s.workerPool.Submit(job)
	return jobID, nil
}

func (s *TaskService) GetWorkerMetrics() queue.WorkerMetrics {
	return s.workerPool.GetMetrics()
}
