CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(255) DEFAULT 'pending',
    due_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT task_status_check CHECK (status IN ('pending', 'in_progress', 'completed'))
);

CREATE INDEX idx_tasks_user_id ON tasks(user_id)
CREATE INDEX idx_tasks_status ON tasks(status)
CREATE INDEX idx_tasks_due_date ON tasks(due_date)