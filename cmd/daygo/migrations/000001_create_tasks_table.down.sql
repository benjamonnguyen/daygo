-- Drop indices first
DROP INDEX IF EXISTS idx_tasks_parent_id;
DROP INDEX IF EXISTS idx_tasks_created_at;
DROP INDEX IF EXISTS idx_tasks_updated_at;
DROP INDEX IF EXISTS idx_tasks_ended_at;
DROP INDEX IF EXISTS idx_tasks_started_at;

-- Drop table
DROP TABLE IF EXISTS tasks;