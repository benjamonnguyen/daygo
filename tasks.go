package daygo

import (
	"context"
	"time"
)

type TaskRepo interface {
	GetTask(context.Context, int) (ExistingTaskRecord, error)
	GetTasks(context.Context, []any) ([]ExistingTaskRecord, error)
	GetByParentID(context.Context, int) ([]ExistingTaskRecord, error)
	GetByStartTime(ctx context.Context, min, max time.Time) ([]ExistingTaskRecord, error)
	CreateTask(context.Context, TaskRecord) (ExistingTaskRecord, error)
	UpdateTask(context.Context, int, UpdatableFields) (ExistingTaskRecord, error)
	DeleteTasks(context.Context, []any) ([]ExistingTaskRecord, error)
}

type TaskRecord struct {
	Name      string
	ParentID  int
	StartedAt time.Time
}

type UpdatableFields struct {
	Name      string
	EndedAt   time.Time
	StartedAt time.Time
}

type ExistingTaskRecord struct {
	TaskRecord
	ID        int
	EndedAt   time.Time
	CreatedAt time.Time
}
