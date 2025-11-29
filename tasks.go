package daygo

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TaskRepo interface {
	// getters
	GetTask(context.Context, uuid.UUID) (ExistingTaskRecord, error)
	GetTasks(context.Context, []any) ([]ExistingTaskRecord, error)
	GetAllTasks(ctx context.Context) ([]ExistingTaskRecord, error)
	GetByParentID(context.Context, uuid.UUID) ([]ExistingTaskRecord, error)
	GetByStartTime(ctx context.Context, min, max time.Time) ([]ExistingTaskRecord, error)
	GetByCreatedTime(ctx context.Context, min, max time.Time) ([]ExistingTaskRecord, error)

	//
	InsertTask(context.Context, TaskRecord) (ExistingTaskRecord, error)
	UpdateTask(context.Context, uuid.UUID, TaskRecord) (ExistingTaskRecord, error)
	DeleteTasks(context.Context, []any) ([]ExistingTaskRecord, error)
}

type TaskRecord struct {
	Name      string
	ParentID  uuid.UUID
	StartedAt time.Time
	EndedAt   time.Time
	QueuedAt  time.Time
}

type ExistingTaskRecord struct {
	TaskRecord
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}
