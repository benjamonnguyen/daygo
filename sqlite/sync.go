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
	SelectAllSyncSessions = "SELECT id, server_url, status, error, to_server_sync_count, from_server_sync_count, created_at FROM sync_sessions"
)

type syncSessionEntity struct {
	ID                  int
	ServerURL           string
	Status              int
	Error               sql.NullString
	ToServerSyncCount   sql.NullInt64
	FromServerSyncCount sql.NullInt64
	CreatedAt           int64
}

// syncSessionRepo
type syncSessionRepo struct {
	dbGetter txStdLib.DBGetter
	l        daygo.Logger
}

var _ daygo.SyncSessionRepo = (*syncSessionRepo)(nil)

func NewSyncSessionRepo(dbGetter txStdLib.DBGetter, logger daygo.Logger) daygo.SyncSessionRepo {
	return &syncSessionRepo{
		l:        logger,
		dbGetter: dbGetter,
	}
}

func (r *syncSessionRepo) GetSession(ctx context.Context, id int) (daygo.ExistingSyncSessionRecord, error) {
	if id == 0 {
		return daygo.ExistingSyncSessionRecord{}, fmt.Errorf("provide id")
	}

	db := r.dbGetter(ctx)
	row := db.QueryRowContext(
		ctx,
		fmt.Sprintf("%s WHERE id=?", SelectAllSyncSessions), id,
	)

	return extractSyncSession(row)
}

func (r *syncSessionRepo) GetLastSession(ctx context.Context, serverURL string, status daygo.SyncStatus) (daygo.ExistingSyncSessionRecord, error) {
	if serverURL == "" {
		return daygo.ExistingSyncSessionRecord{}, fmt.Errorf("provide serverURL")
	}

	db := r.dbGetter(ctx)
	row := db.QueryRowContext(
		ctx,
		fmt.Sprintf("%s WHERE server_url=? AND status=? ORDER BY created_at DESC LIMIT 1", SelectAllSyncSessions),
		serverURL, status,
	)

	return extractSyncSession(row)
}

func (r *syncSessionRepo) InsertSession(ctx context.Context, session daygo.SyncSessionRecord) (daygo.ExistingSyncSessionRecord, error) {
	if session.ServerURL == "" {
		return daygo.ExistingSyncSessionRecord{}, fmt.Errorf("provide required field 'ServerURL'")
	}

	db := r.dbGetter(ctx)
	now := time.Now()

	existingRecord := daygo.ExistingSyncSessionRecord{
		SyncSessionRecord: session,
		CreatedAt:         now,
	}
	e := mapToSyncSessionEntity(existingRecord)

	query := `INSERT INTO sync_sessions (server_url, status, error, to_server_sync_count, from_server_sync_count, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	r.l.Debug("creating sync session", "query", query, "entity", e)
	result, err := db.ExecContext(ctx, query, e.ServerURL, e.Status, e.Error, e.ToServerSyncCount, e.FromServerSyncCount, e.CreatedAt)
	if err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}

	// Get the auto-increment ID and convert to UUID
	insertedID, err := result.LastInsertId()
	if err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}
	existingRecord.ID = int(insertedID)

	return existingRecord, nil
}

func (r *syncSessionRepo) UpdateSession(ctx context.Context, id int, updated daygo.SyncSessionRecord) (daygo.ExistingSyncSessionRecord, error) {
	existing, err := r.GetSession(ctx, id)
	if err != nil {
		return existing, err
	}

	query := "UPDATE sync_sessions SET server_url = ?, status = ?, error = ?, to_server_sync_count = ?, from_server_sync_count = ? WHERE id = ?"
	existing.SyncSessionRecord = updated
	e := mapToSyncSessionEntity(existing)

	r.l.Debug("updating sync session", "query", query, "entity", e)
	if _, err := r.dbGetter(ctx).ExecContext(
		ctx,
		query,
		e.ServerURL, e.Status, e.Error, e.ToServerSyncCount, e.FromServerSyncCount, e.ID,
	); err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}

	return existing, nil
}

func (r *syncSessionRepo) DeleteSession(ctx context.Context, id int) (daygo.ExistingSyncSessionRecord, error) {
	existing, err := r.GetSession(ctx, id)
	if err != nil {
		return existing, err
	}

	query := "DELETE FROM sync_sessions WHERE id = ?"
	r.l.Debug("deleting sync session", "query", query, "id", id)
	if _, err := r.dbGetter(ctx).ExecContext(ctx, query, id); err != nil {
		return daygo.ExistingSyncSessionRecord{}, err
	}

	return existing, nil
}

func extractSyncSession(s scannable) (daygo.ExistingSyncSessionRecord, error) {
	var e syncSessionEntity
	if err := s.Scan(&e.ID, &e.ServerURL, &e.Status, &e.Error, &e.ToServerSyncCount, &e.FromServerSyncCount, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return daygo.ExistingSyncSessionRecord{}, fmt.Errorf("failed to extract sync session: %w", ErrNotFound)
		}
		return daygo.ExistingSyncSessionRecord{}, err
	}

	return mapToExistingSyncSessionRecord(e), nil
}

func mapToSyncSessionEntity(session daygo.ExistingSyncSessionRecord) syncSessionEntity {
	var e syncSessionEntity
	e.ServerURL = session.ServerURL
	e.Status = int(session.Status)
	e.CreatedAt = session.CreatedAt.Unix()

	// Handle Error as nullable string
	if session.Error != "" {
		e.Error = sql.NullString{
			Valid:  true,
			String: session.Error,
		}
	}

	// Handle ToServerSyncCount as nullable int64
	if session.ToServerSyncCount != nil {
		e.ToServerSyncCount = sql.NullInt64{
			Valid: true,
			Int64: int64(*session.ToServerSyncCount),
		}
	}

	// Handle FromServerSyncCount as nullable int64
	if session.FromServerSyncCount != nil {
		e.FromServerSyncCount = sql.NullInt64{
			Valid: true,
			Int64: int64(*session.FromServerSyncCount),
		}
	}

	return e
}

func mapToExistingSyncSessionRecord(e syncSessionEntity) daygo.ExistingSyncSessionRecord {
	// Handle Error as string
	var errorStr string
	if e.Error.Valid {
		errorStr = e.Error.String
	}

	// Handle ToServerSyncCount as *int
	var toServerSyncCount *int
	if e.ToServerSyncCount.Valid {
		val := int(e.ToServerSyncCount.Int64)
		toServerSyncCount = &val
	}

	// Handle FromServerSyncCount as *int
	var fromServerSyncCount *int
	if e.FromServerSyncCount.Valid {
		val := int(e.FromServerSyncCount.Int64)
		fromServerSyncCount = &val
	}

	return daygo.ExistingSyncSessionRecord{
		ID:        e.ID,
		CreatedAt: time.Unix(e.CreatedAt, 0).Local(),
		SyncSessionRecord: daygo.SyncSessionRecord{
			ServerURL:           e.ServerURL,
			Status:              daygo.SyncStatus(e.Status),
			Error:               errorStr,
			ToServerSyncCount:   toServerSyncCount,
			FromServerSyncCount: fromServerSyncCount,
		},
	}
}
