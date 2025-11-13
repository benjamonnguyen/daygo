package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/benjamonnguyen/daygo"
)

type TaskSvc interface {
	StartTask(context.Context, startTaskRequest) (daygo.ExistingTaskRecord, error)
	EndTask(ctx context.Context, id int) (daygo.ExistingTaskRecord, error)
	DeleteTask(ctx context.Context, id int) ([]daygo.ExistingTaskRecord, error)
	RenameTask(ctx context.Context, id int, newName string) (daygo.ExistingTaskRecord, error)
	DequeueTask(ctx context.Context) (daygo.ExistingTaskRecord, error)
	QueueTask(context.Context, queueTaskRequest) (daygo.ExistingTaskRecord, error)
	PeekNextTask(context.Context) (daygo.ExistingTaskRecord, error)
	SkipTask(ctx context.Context, id int) error
}

// models

type Task struct {
	daygo.ExistingTaskRecord
	Notes      []Note
	IsTerminal bool
}

type Note daygo.ExistingTaskRecord

func (t *Task) IsPending() bool {
	return t != nil && t.ID != 0 && t.EndedAt.IsZero()
}

func (t Task) LastNote() *Note {
	if len(t.Notes) > 0 {
		return &t.Notes[len(t.Notes)-1]
	}
	return nil
}

func (t Task) Render(timeFormat string) (string, int) {
	const minLineWidth = 20
	maxItemWidth := len(t.Name)
	var notes []string
	for _, note := range t.Notes {
		if len(note.Name) > maxItemWidth {
			maxItemWidth = len(note.Name)
		}
		notes = append(notes, note.Render(timeFormat))
	}

	l := maxItemWidth + 10
	l = max(minLineWidth, l)

	forDisplay := formatForDisplay(t.ExistingTaskRecord, timeFormat)
	taskLine := fmt.Sprintf("%s %s%c", forDisplay, line(l-len(forDisplay)), tailDown)
	lines := []string{
		taskLine,
	}
	lines = append(lines, notes...)
	if t.IsTerminal {
		endTime := t.EndedAt.Format(timeFormat)
		lines = append(lines, fmt.Sprintf(
			"[%s] %s%c",
			endTime,
			line(l-len(endTime)-2),
			tailUp,
		))
	} else if !t.IsPending() {
		lines = append(lines, fmt.Sprintf("%s%c", line(l+1), tailUp))
	}

	return strings.Join(lines, "\n"), len(lines)
}

func (n Note) Render(timeFormat string) string {
	return formatForDisplay(daygo.ExistingTaskRecord(n), timeFormat)
}

// request objects
type startTaskRequest struct {
	ID       int // if ID is provided, all other fields are ignored
	Name     string
	ParentID int
}

type queueTaskRequest struct {
	Name string
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

// SkipTask replaces provided task with fresh copy
func (s *taskSvc) SkipTask(ctx context.Context, id int) error {
	deleted, err := s.repo.DeleteTasks(ctx, []any{id})
	if err != nil {
		return err
	}
	if len(deleted) == 0 {
		return fmt.Errorf("task not found with id: %d", id)
	}

	og := deleted[0]
	og.StartedAt = time.Time{}
	_, err = s.repo.CreateTask(ctx, og.TaskRecord)
	return err
}

func (s *taskSvc) StartTask(ctx context.Context, req startTaskRequest) (daygo.ExistingTaskRecord, error) {
	now := time.Now().Local()
	if req.ID != 0 {
		return s.repo.UpdateTask(ctx, req.ID, daygo.UpdatableFields{
			StartedAt: now,
		})
	}

	return s.repo.CreateTask(ctx, daygo.TaskRecord{
		Name:      req.Name,
		ParentID:  req.ParentID,
		StartedAt: now,
	})
}

func (s *taskSvc) EndTask(ctx context.Context, id int) (daygo.ExistingTaskRecord, error) {
	return s.repo.UpdateTask(ctx, id, daygo.UpdatableFields{
		EndedAt: time.Now().Local(),
	})
}

func (s *taskSvc) DeleteTask(ctx context.Context, id int) ([]daygo.ExistingTaskRecord, error) {
	res, err := s.repo.DeleteTasks(ctx, []any{id})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *taskSvc) RenameTask(ctx context.Context, id int, newName string) (daygo.ExistingTaskRecord, error) {
	return s.repo.UpdateTask(ctx, id, daygo.UpdatableFields{
		Name: newName,
	})
}

func (s *taskSvc) QueueTask(ctx context.Context, req queueTaskRequest) (daygo.ExistingTaskRecord, error) {
	return s.repo.CreateTask(ctx, daygo.TaskRecord{
		Name: req.Name,
	})
}

func (s *taskSvc) DequeueTask(ctx context.Context) (daygo.ExistingTaskRecord, error) {
	next, err := s.PeekNextTask(ctx)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	if next.ID == 0 {
		return daygo.ExistingTaskRecord{}, fmt.Errorf("task queue is empty")
	}

	return s.StartTask(ctx, startTaskRequest{
		ID: next.ID,
	})
}

func (s *taskSvc) PeekNextTask(ctx context.Context) (daygo.ExistingTaskRecord, error) {
	tasks, err := s.repo.GetByStartTime(ctx, time.Time{}, time.Time{})
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}
	if len(tasks) == 0 {
		return daygo.ExistingTaskRecord{}, nil
	}

	earliest := tasks[0]
	for _, task := range tasks[1:] {
		if task.CreatedAt.Compare(earliest.CreatedAt) < 0 {
			earliest = task
		}
	}

	return earliest, nil
}
