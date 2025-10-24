package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/benjamonnguyen/daygo"
)

// reposervice
type reposervice struct {
	db *sql.DB
}

func NewReposervice(db *sql.DB) *reposervice {
	return &reposervice{
		db: db,
	}
}

func (a *reposervice) CreateTask(ctx context.Context, req daygo.CreateTaskRequest) (daygo.Task, error) {
	if req.Text == "" {
		return daygo.Task{}, fmt.Errorf("provide required field 'Text'")
	}

	res, err := a.db.ExecContext(
		ctx,
		`INSERT INTO tasks (text, priority) VALUES (?,?);`, req.Text, req.Priority,
	)
	if err != nil {
		return daygo.Task{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return daygo.Task{}, err
	}
	return daygo.Task{
		ID:       int(id),
		Text:     req.Text,
		Priority: req.Priority,
	}, nil
}

func (a *reposervice) UpdateTask(ctx context.Context, req daygo.UpdateTaskRequest) (daygo.Task, error) {
	// TODO UpdateTask
	return daygo.Task{}, nil
}

func (a *reposervice) StartSession(ctx context.Context, req daygo.StartSessionRequest) (daygo.Session, error) {
	// TODO UpdateTask
	return daygo.Session{}, nil
}

func (a *reposervice) EndSession(ctx context.Context, req daygo.EndSessionRequest) (daygo.Session, error) {
	// TODO UpdateTask
	return daygo.Session{}, nil
}
