package main

import (
	"context"
	"time"

	"github.com/benjamonnguyen/daygo"
	"github.com/google/uuid"
)

type TaskSvc interface {
	UpsertTask(context.Context, Task) (daygo.ExistingTaskRecord, error)
	DeleteTask(ctx context.Context, id uuid.UUID) ([]daygo.ExistingTaskRecord, error)
	GetAllTasks(ctx context.Context) ([]Task, error)

	// sync
	GetTasksToSync(ctx context.Context, serverURL string) ([]daygo.ExistingTaskRecord, error)
	GetLastSuccessfulSync(ctx context.Context, serverURL string) (daygo.ExistingSyncSessionRecord, error)
}

// impl
type taskSvc struct {
	taskRepo        daygo.TaskRepo
	syncSessionRepo daygo.SyncSessionRepo
}

func NewTaskSvc(taskRepo daygo.TaskRepo, syncSessionRepo daygo.SyncSessionRepo) TaskSvc {
	return &taskSvc{
		taskRepo:        taskRepo,
		syncSessionRepo: syncSessionRepo,
	}
}

func (s *taskSvc) UpsertTask(ctx context.Context, t Task) (daygo.ExistingTaskRecord, error) {
	// insert
	if t.ID == uuid.Nil {
		inserted, err := s.taskRepo.InsertTask(ctx, t.TaskRecord)
		if err != nil {
			return daygo.ExistingTaskRecord{}, err
		}
		if err := s.createNotes(ctx, t.Notes); err != nil {
			return daygo.ExistingTaskRecord{}, err
		}
		return inserted, nil
	}
	// update
	updated, err := s.taskRepo.UpdateTask(ctx, t.ID, t.TaskRecord)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}
	if err := s.createNotes(ctx, t.Notes); err != nil {
		return daygo.ExistingTaskRecord{}, err
	}
	return updated, nil
}

func (s *taskSvc) createNotes(ctx context.Context, notes []Note) error {
	for _, n := range notes {
		_, err := s.taskRepo.InsertTask(ctx, daygo.TaskRecord(n))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *taskSvc) GetAllTasks(ctx context.Context) ([]Task, error) {
	records, err := s.taskRepo.GetByStartTime(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(records))
	for _, r := range records {
		tasks = append(tasks, TaskFromRecord(r))
	}

	return tasks, nil
}

func (s *taskSvc) DeleteTask(ctx context.Context, id uuid.UUID) ([]daygo.ExistingTaskRecord, error) {
	res, err := s.taskRepo.DeleteTasks(ctx, []any{id.String()})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *taskSvc) GetTasksToSync(ctx context.Context, serverURL string) ([]daygo.ExistingTaskRecord, error) {
	lastSyncSession, err := s.syncSessionRepo.GetLastSession(ctx, serverURL, daygo.SyncStatusPartial)
	if err != nil {
		return nil, err
	}
	if lastSyncSession.ID == 0 {
		return s.taskRepo.GetAllTasks(ctx)
	}

	lastSyncTime := lastSyncSession.CreatedAt
	tasks, err := s.taskRepo.GetByUpdateTime(ctx, lastSyncTime, time.Time{})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *taskSvc) GetLastSuccessfulSync(ctx context.Context, serverURL string) (daygo.ExistingSyncSessionRecord, error) {
	return s.syncSessionRepo.GetLastSession(ctx, serverURL, daygo.SyncStatusSuccess)
}
