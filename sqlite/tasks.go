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
	"github.com/google/uuid"
)

const (
	SelectAll = "SELECT id, name, started_at, ended_at, parent_id, created_at, updated_at, queued_at FROM tasks"
)

var ErrNotFound = errors.New("not found")

type taskEntity struct {
	ID        string
	Name      string
	StartedAt sql.NullInt64
	EndedAt   sql.NullInt64
	CreatedAt int64
	UpdatedAt int64
	ParentID  sql.NullString
	QueuedAt  sql.NullInt64
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

func (r *taskRepo) GetTask(ctx context.Context, id uuid.UUID) (daygo.ExistingTaskRecord, error) {
	if id == uuid.Nil {
		return daygo.ExistingTaskRecord{}, fmt.Errorf("provide id")
	}

	db := r.dbGetter(ctx)
	row := db.QueryRowContext(
		ctx,
		fmt.Sprintf("%s WHERE id=?", SelectAll), id.String(),
	)

	return extractTask(row)
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
		return nil, fmt.Errorf("expected %d tasks, got %d: %+v: %w", len(ids), len(tasks), tasks, ErrNotFound)
	}
	return tasks, nil
}

func (r *taskRepo) GetByStartTime(ctx context.Context, min, max time.Time) ([]daygo.ExistingTaskRecord, error) {
	query := SelectAll
	var args []any
	if !min.IsZero() && !max.IsZero() {
		query += " WHERE started_at BETWEEN ? AND ?"
		args = append(args, min.Unix(), max.Unix())
	} else if !min.IsZero() {
		query += " WHERE created_at >= ?"
		args = append(args, min.Unix())
	} else if !max.IsZero() {
		query += " WHERE created_at <= ?"
		args = append(args, max.Unix())
	} else {
		query += " WHERE started_at ISNULL"
	}

	db := r.dbGetter(ctx)
	r.l.Debug("GetByStartTime", "query", query, "args", args)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	tasks, err := extractTasks(rows)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepo) GetByParentID(ctx context.Context, parentID uuid.UUID) ([]daygo.ExistingTaskRecord, error) {
	if parentID == uuid.Nil {
		return nil, fmt.Errorf("provide parentID")
	}

	db := r.dbGetter(ctx)
	rows, err := db.QueryContext(
		ctx,
		fmt.Sprintf("%s WHERE parent_id=?", SelectAll), parentID.String(),
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

func (r *taskRepo) GetByCreateTime(ctx context.Context, min, max time.Time) ([]daygo.ExistingTaskRecord, error) {
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

func (r *taskRepo) GetByUpdateTime(ctx context.Context, min, max time.Time) ([]daygo.ExistingTaskRecord, error) {
	query := SelectAll
	var args []any

	if !min.IsZero() && !max.IsZero() {
		query += " WHERE updated_at BETWEEN ? AND ?"
		args = append(args, min.Unix(), max.Unix())
	} else if !min.IsZero() {
		query += " WHERE updated_at >= ?"
		args = append(args, min.Unix())
	} else if !max.IsZero() {
		query += " WHERE updated_at <= ?"
		args = append(args, max.Unix())
	}

	db := r.dbGetter(ctx)
	r.l.Debug("GetByUpdateTime", "query", query, "args", args)
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
	if err := s.Scan(&e.ID, &e.Name, &e.StartedAt, &e.EndedAt, &e.ParentID, &e.CreatedAt, &e.UpdatedAt, &e.QueuedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return daygo.ExistingTaskRecord{}, ErrNotFound
		}
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

	existingRecord := daygo.ExistingTaskRecord{
		TaskRecord: task,
		ID:         uuid.New(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	e := mapToTaskEntity(existingRecord)

	args := []any{
		e.ID,
		e.Name,
		e.ParentID,
		e.StartedAt,
		e.EndedAt,
		e.CreatedAt,
		e.UpdatedAt,
		e.QueuedAt,
	}
	query := "INSERT INTO tasks (id, name, parent_id, started_at, ended_at, created_at, updated_at, queued_at) VALUES " + generateParameters(len(args))
	r.l.Debug("creating task", "query", query, "args", args)
	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	return existingRecord, nil
}

func (r *taskRepo) UpdateTask(ctx context.Context, id uuid.UUID, updated daygo.TaskRecord) (daygo.ExistingTaskRecord, error) {
	existing, err := r.GetTask(ctx, id)
	if err != nil {
		return existing, err
	}

	existing.TaskRecord = updated
	existing.UpdatedAt = time.Now()
	e := mapToTaskEntity(existing)

	query := "UPDATE tasks SET name = ?, started_at = ?, ended_at = ?, queued_at = ?, updated_at = ? WHERE id = ?"
	args := []any{
		e.Name,
		e.StartedAt,
		e.EndedAt,
		e.QueuedAt,
		e.UpdatedAt,
		e.ID,
	}
	r.l.Debug("updating task", "query", query, "args", args)
	_, err = r.dbGetter(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return daygo.ExistingTaskRecord{}, err
	}

	return existing, nil
}

func (r *taskRepo) DeleteTasks(ctx context.Context, ids []any) ([]daygo.ExistingTaskRecord, error) {
	toDelete, err := r.GetTasks(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(toDelete) != len(ids) {
		return nil, fmt.Errorf("expected %d existing tasks, found %d: %w", len(ids), len(toDelete), ErrNotFound)
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
	e.ID = task.ID.String()

	// Handle ParentID as nullable string
	if task.ParentID != uuid.Nil {
		e.ParentID = sql.NullString{
			Valid:  true,
			String: task.ParentID.String(),
		}
	}

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
	if !task.QueuedAt.IsZero() {
		e.QueuedAt = sql.NullInt64{
			Valid: true,
			Int64: task.QueuedAt.Unix(),
		}
	}
	return e
}

func mapToExistingTaskRecord(e taskEntity) daygo.ExistingTaskRecord {
	var startedAt, endedAt, queuedAt time.Time
	if e.StartedAt.Valid {
		startedAt = time.Unix(e.StartedAt.Int64, 0).Local()
	}
	if e.EndedAt.Valid {
		endedAt = time.Unix(e.EndedAt.Int64, 0).Local()
	}
	if e.QueuedAt.Valid {
		queuedAt = time.Unix(e.QueuedAt.Int64, 0).Local()
	}

	// Parse UUID for ID
	id, _ := uuid.Parse(e.ID)

	// Parse ParentID as UUID if present
	var parentID uuid.UUID
	if e.ParentID.Valid && e.ParentID.String != "" {
		parentID, _ = uuid.Parse(e.ParentID.String)
	}

	return daygo.ExistingTaskRecord{
		ID:        id,
		CreatedAt: time.Unix(e.CreatedAt, 0).Local(),
		UpdatedAt: time.Unix(e.UpdatedAt, 0).Local(),
		TaskRecord: daygo.TaskRecord{
			Name:      e.Name,
			ParentID:  parentID,
			StartedAt: startedAt,
			EndedAt:   endedAt,
			QueuedAt:  queuedAt,
		},
	}
}
