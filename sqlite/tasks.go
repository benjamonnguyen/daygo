// Package sqlite implements reposervice interfaces
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	txStdLib "github.com/Thiht/transactor/stdlib"

	"github.com/benjamonnguyen/daygo"
)

const (
	SelectAll = "SELECT id, name, started_at, ended_at, parent_id, created_at, updated_at FROM tasks"
)

var ErrNotFound = errors.New("not found")

type taskEntity struct {
	ID        int
	Name      string
	StartedAt sql.NullInt64
	EndedAt   sql.NullInt64
	CreatedAt int64
	UpdatedAt int64
	ParentID  int
}

// taskRepo
type taskRepo struct {
	dbGetter txStdLib.DBGetter
	l        daygo.Logger
}

var _ daygo.TaskRepo = (*taskRepo)(nil)

func NewTaskRepo(dbGetter txStdLib.DBGetter, logger daygo.Logger) daygo.TaskRepo {
	return &taskRepo{
		l:        logger,
		dbGetter: dbGetter,
	}
}

func (r *taskRepo) GetTask(ctx context.Context, id int) (daygo.ExistingTaskRecord, error) {
	if id == 0 {
		return daygo.ExistingTaskRecord{}, fmt.Errorf("provide id")
	}

	db := r.dbGetter(ctx)
	row := db.QueryRowContext(
		ctx,
		fmt.Sprintf("%s WHERE id=?", SelectAll), id,
	)

	task, err := extractTask(row)
	if err != nil {
		return task, err
	}
	if task.ID == 0 {
		return task, ErrNotFound
	}
	return task, nil
}

func (r taskRepo) GetAllTasks(ctx context.Context) ([]daygo.ExistingTaskRecord, error) {
	db := r.dbGetter(ctx)
	rows, err := db.QueryContext(ctx, SelectAll)
	if err != nil {
		return nil, err
	}

	return extractTasks(rows)
}

func (r taskRepo) GetTasks(ctx context.Context, ids []any) ([]daygo.ExistingTaskRecord, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("provide ids")
	}

	db := r.dbGetter(ctx)
	query := fmt.Sprintf("%s WHERE id IN %s", SelectAll, generateParameters(len(ids)))
	r.l.Debug("getting tasks", "query", query)
	rows, err := db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}

	tasks, err := extractTasks(rows)
	if err != nil {
		return nil, err
	}
	if len(tasks) != len(ids) {
		return nil, fmt.Errorf("expected %d tasks, got %d: %+v", len(ids), len(tasks), tasks)
	}
	return tasks, nil
}

func (r *taskRepo) GetByStartTime(ctx context.Context, min, max time.Time) ([]daygo.ExistingTaskRecord, error) {
	query := SelectAll
	if min.IsZero() && max.IsZero() {
		query += " WHERE started_at ISNULL"
	}

	db := r.dbGetter(ctx)
	r.l.Debug("GetByStartTime", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return extractTasks(rows)
}

func (r *taskRepo) GetByParentID(ctx context.Context, parentID int) ([]daygo.ExistingTaskRecord, error) {
	if parentID == 0 {
		return nil, fmt.Errorf("provide parentID")
	}

	db := r.dbGetter(ctx)
	rows, err := db.QueryContext(
		ctx,
		fmt.Sprintf("%s WHERE parent_id=?", SelectAll), parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var subtasks []daygo.ExistingTaskRecord
	for rows.Next() {
		subtask, err := extractTask(rows)
		if err != nil {
			return nil, err
		}
		subtasks = append(subtasks, subtask)
	}

	return subtasks, nil
}

func (r *taskRepo) GetByCreatedTime(ctx context.Context, min, max time.Time) ([]daygo.ExistingTaskRecord, error) {
	query := SelectAll
	var args []any

	if !min.IsZero() && !max.IsZero() {
		query += " WHERE created_at BETWEEN ? AND ?"
		args = append(args, min.Unix(), max.Unix())
	} else if !min.IsZero() {
		query += " WHERE created_at >= ?"
		args = append(args, min.Unix())
	} else if !max.IsZero() {
		query += " WHERE created_at <= ?"
		args = append(args, max.Unix())
	}

	db := r.dbGetter(ctx)
	r.l.Debug("GetByCreatedTime", "query", query, "args", args)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return extractTasks(rows)
}

func extractTasks(rows *sql.Rows) ([]daygo.ExistingTaskRecord, error) {
	var tasks []daygo.ExistingTaskRecord
	for rows.Next() {
		task, err := extractTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func extractTask(s scannable) (daygo.ExistingTaskRecord, error) {
	var e taskEntity
	if err := s.Scan(&e.ID, &e.Name, &e.StartedAt, &e.EndedAt, &e.ParentID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	return mapToExistingTaskRecord(e), nil
}

func (r *taskRepo) InsertTask(ctx context.Context, task daygo.TaskRecord) (daygo.ExistingTaskRecord, error) {
	if task.Name == "" {
		return daygo.ExistingTaskRecord{}, fmt.Errorf("provide required field 'Name'")
	}

	db := r.dbGetter(ctx)
	now := time.Now()
	e := mapToTaskEntity(daygo.ExistingTaskRecord{
		TaskRecord: task,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	query := `INSERT INTO tasks (name, parent_id, started_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	r.l.Debug("creating task", "query", query, "entity", e)
	res, err := db.ExecContext(ctx, query, e.Name, e.ParentID, e.StartedAt, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}
	created, err := r.GetTask(ctx, int(id))
	r.l.Debug("created task", "task", created)
	return created, err
}

func (r *taskRepo) UpdateTask(ctx context.Context, id int, updated daygo.TaskRecord) (daygo.ExistingTaskRecord, error) {
	existing, err := r.GetTask(ctx, id)
	if err != nil {
		return existing, err
	}

	existing.TaskRecord = updated
	existing.UpdatedAt = time.Now()
	e := mapToTaskEntity(existing)

	db := r.dbGetter(ctx)
	_, err = db.ExecContext(
		ctx,
		`UPDATE tasks
		SET ended_at = ?, name = ?, started_at = ?, updated_at = ?
		WHERE id = ?`,
		e.EndedAt, e.Name, e.StartedAt, e.UpdatedAt, e.ID,
	)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	r.l.Debug("updated task", "task", existing)
	return existing, nil
}

func (r *taskRepo) DeleteTasks(ctx context.Context, ids []any) ([]daygo.ExistingTaskRecord, error) {
	toDelete, err := r.GetTasks(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(toDelete) != len(ids) {
		return nil, fmt.Errorf("expected %d existing tasks, found %d", len(ids), len(toDelete))
	}

	db := r.dbGetter(ctx)
	query := fmt.Sprintf("DELETE FROM tasks WHERE id IN %s", generateParameters(len(ids)))
	r.l.Debug("deleting tasks", "query", query, "ids", ids)
	if _, err := db.ExecContext(ctx, query, ids...); err != nil {
		return nil, err
	}

	// cascade delete subtasks
	query = fmt.Sprintf("DELETE FROM tasks WHERE parent_id IN %s", generateParameters(len(ids)))
	r.l.Debug("deleting subtasks", "query", query)
	if _, err := db.ExecContext(ctx, query, ids...); err != nil {
		return nil, err
	}

	return toDelete, nil
}

func mapToTaskEntity(task daygo.ExistingTaskRecord) taskEntity {
	var e taskEntity
	e.Name = task.Name
	e.CreatedAt = task.CreatedAt.Unix()
	e.UpdatedAt = task.UpdatedAt.Unix()
	e.ID = task.ID
	e.ParentID = task.ParentID
	if !task.StartedAt.IsZero() {
		e.StartedAt = sql.NullInt64{
			Valid: true,
			Int64: task.StartedAt.Unix(),
		}
	}
	if !task.EndedAt.IsZero() {
		e.EndedAt = sql.NullInt64{
			Valid: true,
			Int64: task.EndedAt.Unix(),
		}
	}
	return e
}

func mapToExistingTaskRecord(e taskEntity) daygo.ExistingTaskRecord {
	var startedAt, endedAt time.Time
	if e.StartedAt.Valid {
		startedAt = time.Unix(e.StartedAt.Int64, 0).Local()
	}
	if e.EndedAt.Valid {
		endedAt = time.Unix(e.EndedAt.Int64, 0).Local()
	}

	return daygo.ExistingTaskRecord{
		ID:        e.ID,
		CreatedAt: time.Unix(e.CreatedAt, 0).Local(),
		UpdatedAt: time.Unix(e.UpdatedAt, 0).Local(),
		TaskRecord: daygo.TaskRecord{
			Name:      e.Name,
			ParentID:  e.ParentID,
			StartedAt: startedAt,
			EndedAt:   endedAt,
		},
	}
}
