-- Add updated_at column to tasks table with default value 0
ALTER TABLE tasks ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;