package main

import (
	"context"
	"errors"
	"time"

	"github.com/Thiht/transactor"
	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/sqlite"
	"github.com/google/uuid"
)

type TaskSvc interface {
	UpsertTask(context.Context, Task) (Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID) ([]daygo.ExistingTaskRecord, error)
	QueueTask(context.Context, Task) (Task, error)
	GetPendingTasks(ctx context.Context) ([]Task, error)

	// sync
	GetTasksToSync(ctx context.Context, serverURL string) ([]daygo.ExistingTaskRecord, error)
	GetLastSuccessfulSync(ctx context.Context, serverURL string) (daygo.ExistingSyncSessionRecord, error)
	UpsertSyncSession(context.Context, int, daygo.SyncSessionRecord) (daygo.ExistingSyncSessionRecord, error)
	SyncTasks(ctx context.Context, serverTasks []daygo.ExistingTaskRecord) ([]Task, []error)
}

// impl
type taskSvc struct {
	logger          daygo.Logger
	transactor      transactor.Transactor
	taskRepo        daygo.TaskRepo
	syncSessionRepo daygo.SyncSessionRepo
}

func NewTaskSvc(transactor transactor.Transactor, logger daygo.Logger, taskRepo daygo.TaskRepo, syncSessionRepo daygo.SyncSessionRepo) TaskSvc {
	return &taskSvc{
		logger:          logger,
		transactor:      transactor,
		taskRepo:        taskRepo,
		syncSessionRepo: syncSessionRepo,
	}
}

func (s *taskSvc) QueueTask(ctx context.Context, t Task) (Task, error) {
	t.QueuedAt = time.Now()
	t.StartedAt = time.Time{}
	return s.UpsertTask(ctx, t)
}

func (s *taskSvc) SyncTasks(ctx context.Context, serverTasks []daygo.ExistingTaskRecord) ([]Task, []error) {
	// Collect serverTaskIDs
	serverTaskIDs := make([]any, 0, len(serverTasks))
	for _, serverTask := range serverTasks {
		serverTaskIDs = append(serverTaskIDs, serverTask.ID.String())
	}

	// Get existing client tasks
	clientTasks, err := s.taskRepo.GetTasks(ctx, serverTaskIDs)
	if err != nil && !errors.Is(err, sqlite.ErrNotFound) {
		return nil, []error{err}
	}

	// Create a map for quick lookup of client tasks by ID
	clientTaskMap := make(map[uuid.UUID]daygo.ExistingTaskRecord)
	for _, clientTask := range clientTasks {
		clientTaskMap[clientTask.ID] = clientTask
	}

	var upserted []Task
	var errs []error
	for _, serverTask := range serverTasks {
		clientTask, exists := clientTaskMap[serverTask.ID]
		if !exists || serverTask.UpdatedAt.After(clientTask.UpdatedAt) {
			u, err := s.UpsertTask(ctx, TaskFromRecord(serverTask))
			if err != nil {
				errs = append(errs, err)
			} else {
				upserted = append(upserted, u)
			}
		}
	}
	return upserted, errs
}

func (s *taskSvc) UpsertTask(ctx context.Context, t Task) (Task, error) {
	var res daygo.ExistingTaskRecord
	// update
	if t.ID != uuid.Nil {
		updated, err := s.taskRepo.UpdateTask(ctx, t.ID, t.TaskRecord)
		if err != nil {
			if !errors.Is(err, sqlite.ErrNotFound) {
				return Task{}, err
			}
		}
		res = updated
	}
	// insert
	if res.ID == uuid.Nil {
		inserted, err := s.taskRepo.InsertTask(ctx, t.TaskRecord)
		if err != nil {
			return Task{}, err
		}
		res = inserted
	}

	// notes
	if err := s.createNotes(ctx, res.ID, t.Notes); err != nil {
		return Task{}, err
	}

	return TaskFromRecord(res), nil
}

func (s *taskSvc) createNotes(ctx context.Context, parentID uuid.UUID, notes []Note) error {
	for _, n := range notes {
		n.ParentID = parentID
		_, err := s.taskRepo.InsertTask(ctx, daygo.TaskRecord(n))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *taskSvc) GetPendingTasks(ctx context.Context) ([]Task, error) {
	// get tasks not started yet and is queued up
	records, err := s.taskRepo.GetByStartTime(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(records))
	for _, r := range records {
		if !r.UpdatedAt.IsZero() {
			tasks = append(tasks, TaskFromRecord(r))
		}
	}

	return tasks, nil
}

func (s *taskSvc) DeleteTask(ctx context.Context, id uuid.UUID) ([]daygo.ExistingTaskRecord, error) {
	res, err := s.taskRepo.DeleteTasks(ctx, []any{id})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *taskSvc) GetTasksToSync(ctx context.Context, serverURL string) ([]daygo.ExistingTaskRecord, error) {
	var lastSync daygo.ExistingSyncSessionRecord
	lastPartialSync, err := s.syncSessionRepo.GetLastSession(ctx, serverURL, daygo.SyncStatusPartial)
	if err != nil {
		if !errors.Is(err, sqlite.ErrNotFound) {
			return nil, err
		}
	} else {
		lastSync = lastPartialSync
	}
	lastSuccessfulSync, err := s.syncSessionRepo.GetLastSession(ctx, serverURL, daygo.SyncStatusSuccess)
	if err != nil {
		if !errors.Is(err, sqlite.ErrNotFound) {
			return nil, err
		}
	} else if lastSuccessfulSync.CreatedAt.After(lastSync.CreatedAt) {
		lastSync = lastSuccessfulSync
	}
	if lastSync.ID == 0 {
		return s.taskRepo.GetAllTasks(ctx)
	}

	return s.taskRepo.GetByUpdateTime(ctx, lastSync.CreatedAt, time.Time{})
}

func (s *taskSvc) GetLastSuccessfulSync(ctx context.Context, serverURL string) (daygo.ExistingSyncSessionRecord, error) {
	session, err := s.syncSessionRepo.GetLastSession(ctx, serverURL, daygo.SyncStatusSuccess)
	if err != nil && !errors.Is(err, sqlite.ErrNotFound) {
		return session, err
	}
	return session, nil
}

func (s *taskSvc) UpsertSyncSession(ctx context.Context, id int, session daygo.SyncSessionRecord) (daygo.ExistingSyncSessionRecord, error) {
	// insert
	if id == 0 {
		inserted, err := s.syncSessionRepo.InsertSession(ctx, session)
		if err != nil {
			return daygo.ExistingSyncSessionRecord{}, err
		}
		return inserted, nil
	}
	// update
	updated, err := s.syncSessionRepo.UpdateSession(ctx, id, session)
	if err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}
	return updated, nil
}

func (s *taskSvc) UpsertSyncSession(ctx context.Context, id int, session daygo.SyncSessionRecord) (daygo.ExistingSyncSessionRecord, error) {
	// insert
	if id == 0 {
		inserted, err := s.syncSessionRepo.InsertSession(ctx, session)
		if err != nil {
			return daygo.ExistingSyncSessionRecord{}, err
		}
		return inserted, nil
	}
	// update
	updated, err := s.syncSessionRepo.UpdateSession(ctx, id, session)
	if err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}
	return updated, nil
}
