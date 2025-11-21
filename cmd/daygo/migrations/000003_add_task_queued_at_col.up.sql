-- Add queued_at column to tasks table
ALTER TABLE tasks ADD COLUMN queued_at INTEGER;

-- Backfill queued_at with started_at values for existing tasks
UPDATE tasks SET queued_at = started_at WHERE queued_at IS NULL;