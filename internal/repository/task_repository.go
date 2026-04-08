package repository

import (
	"context"
	"database/sql"
	"fmt"
	"task-manager/internal/models"
	"time"

	"github.com/google/uuid"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	query := `
        INSERT INTO tasks (user_id, title, description, status, due_date)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at
    `

	err := r.db.QueryRowContext(ctx, query,
		task.UserID, task.Title, task.Description, task.Status, task.DueDate,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Task, error) {
	query := `
        SELECT id, user_id, title, description, status, due_date, created_at, updated_at
        FROM tasks
        WHERE id = $1 AND user_id = $2
    `

	var task models.Task
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&task.ID, &task.UserID, &task.Title, &task.Description,
		&task.Status, &task.DueDate, &task.CreatedAt, &task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task by id: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context, userID uuid.UUID, status *models.TaskStatus, limit, offset int) ([]*models.Task, error) {
	query := `
        SELECT id, user_id, title, description, status, due_date, created_at, updated_at
        FROM tasks
        WHERE user_id = $1
    `
	args := []interface{}{userID}
	argIndex := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID, &task.UserID, &task.Title, &task.Description,
			&task.Status, &task.DueDate, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	query := `
        UPDATE tasks
        SET title = $1, description = $2, status = $3, due_date = $4, updated_at = CURRENT_TIMESTAMP
        WHERE id = $5 AND user_id = $6
        RETURNING updated_at
    `

	err := r.db.QueryRowContext(ctx, query,
		task.Title, task.Description, task.Status, task.DueDate, task.ID, task.UserID,
	).Scan(&task.UpdatedAt)

	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

type OverdueTaskInfo struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Title   string    `json:"title"`
	DueDate time.Time `json:"due_date"`
	Status  string    `json:"status"`
}

func (r *TaskRepository) GetOverdueTasks(ctx context.Context) ([]OverdueTaskInfo, error) {
	query := `
        SELECT id, user_id, title, due_date, status
        FROM tasks
        WHERE due_date < NOW() 
          AND status != 'completed'
          AND status != 'cancelled'
        ORDER BY due_date ASC
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue tasks: %w", err)
	}
	defer rows.Close()

	var tasks []OverdueTaskInfo
	for rows.Next() {
		var task OverdueTaskInfo
		err := rows.Scan(
			&task.ID,
			&task.UserID,
			&task.Title,
			&task.DueDate,
			&task.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan overdue task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	query := `
        UPDATE tasks 
        SET status = $1, updated_at = CURRENT_TIMESTAMP
        WHERE id = $2
    `

	result, err := r.db.ExecContext(ctx, query, status, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

var (
	ErrTaskNotFound = fmt.Errorf("task not found")
)
