package daygo

import (
	"context"
	"time"
)

type CreateTaskRequest struct {
	Text     string
	Priority TaskPriority
}

type UpdateTaskRequest struct {
	Priority    TaskPriority
	CompletedAt time.Time
}

type StartSessionRequest struct {
	TaskID int
}

type EndSessionRequest struct {
	EndedAt time.Time
}

type RepoService interface {
	CreateTask(context.Context, CreateTaskRequest) (Task, error)
	UpdateTask(context.Context, UpdateTaskRequest) (Task, error)
	StartSession(context.Context, StartSessionRequest) (Session, error)
	EndSession(context.Context, EndSessionRequest) (Session, error)
}
