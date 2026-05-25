ALTER TABLE tasks
DROP CONSTRAINT tasks_status_check;

ALTER TABLE tasks
ADD CONSTRAINT tasks_status_check
CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled', 'overdue'));

CREATE INDEX idx_tasks_overdue_check
ON tasks(due_date, status)
WHERE status != 'completed' AND status != 'cancelled';

ALTER TABLE tasks
ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at) WHERE deleted_at IS NULL;