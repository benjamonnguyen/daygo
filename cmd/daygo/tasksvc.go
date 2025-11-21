package main

import (
	"context"
	"time"

	"github.com/benjamonnguyen/daygo"
)

type TaskSvc interface {
	UpsertTask(context.Context, Task) (Task, error)
	DeleteTask(ctx context.Context, id int) ([]daygo.ExistingTaskRecord, error)
	GetAllTasks(ctx context.Context) ([]Task, error)
}

// impl
type taskSvc struct {
	repo daygo.TaskRepo
}

func NewTaskSvc(taskRepo daygo.TaskRepo) TaskSvc {
	return &taskSvc{
		repo: taskRepo,
	}
}

func (s *taskSvc) UpsertTask(ctx context.Context, t Task) (Task, error) {
	// insert
	if t.ID == 0 {
		inserted, err := s.repo.InsertTask(ctx, t.TaskRecord)
		if err != nil {
			return Task{}, err
		}
		if err := s.createNotes(ctx, t.Notes); err != nil {
			return Task{}, err
		}
		return Task{
			ExistingTaskRecord: inserted,
		}, nil
	}
	// update
	updated, err := s.repo.UpdateTask(ctx, t.ID, t.TaskRecord)
	if err != nil {
		return Task{}, err
	}
	if err := s.createNotes(ctx, t.Notes); err != nil {
		return Task{}, err
	}
	return Task{
		ExistingTaskRecord: updated,
	}, nil
}

func (s *taskSvc) createNotes(ctx context.Context, notes []Note) error {
	for _, n := range notes {
		_, err := s.repo.InsertTask(ctx, daygo.TaskRecord(n))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *taskSvc) GetAllTasks(ctx context.Context) ([]Task, error) {
	records, err := s.repo.GetByStartTime(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(records))
	for _, r := range records {
		tasks = append(tasks, Task{
			ExistingTaskRecord: r,
		})
	}

	return tasks, nil
}

func (s *taskSvc) DeleteTask(ctx context.Context, id int) ([]daygo.ExistingTaskRecord, error) {
	res, err := s.repo.DeleteTasks(ctx, []any{id})
	if err != nil {
		return nil, err
	}
	return res, nil
}
