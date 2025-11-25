package daygo

import (
	"context"
	"time"
)

type TaskRepo interface {
	GetTask(context.Context, int) (ExistingTaskRecord, error)
	GetTasks(context.Context, []any) ([]ExistingTaskRecord, error)
	GetAllTasks(ctx context.Context) ([]ExistingTaskRecord, error)
	GetByParentID(context.Context, int) ([]ExistingTaskRecord, error)
	GetByStartTime(ctx context.Context, min, max time.Time) ([]ExistingTaskRecord, error)
	InsertTask(context.Context, TaskRecord) (ExistingTaskRecord, error)
	UpdateTask(context.Context, int, TaskRecord) (ExistingTaskRecord, error)
	DeleteTasks(context.Context, []any) ([]ExistingTaskRecord, error)
}

type TaskRecord struct {
	Name      string
	ParentID  int
	StartedAt time.Time
	EndedAt   time.Time
	QueuedAt  time.Time
}

type ExistingTaskRecord struct {
	TaskRecord
	ID        int
	CreatedAt time.Time
}
