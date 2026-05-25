DROP INDEX IF EXISTS idx_tasks_overdue_check;
DROP INDEX IF EXISTS idx_tasks_deleted_at;

ALTER TABLE tasks DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE tasks
DROP CONSTRAINT tasks_status_check;

ALTER TABLE tasks
    ADD CONSTRAINT tasks_status_check
        CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled'));