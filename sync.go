package daygo

import (
	"context"
	"time"
)

type SyncSessionRepo interface {
	GetSyncSession(ctx context.Context, id int) (ExistingSyncSessionRecord, error)
	GetLatestSession(ctx context.Context, serverURL string) (ExistingSyncSessionRecord, error)
	InsertSyncSession(ctx context.Context, session SyncSessionRecord) (ExistingSyncSessionRecord, error)
	UpdateSyncSession(ctx context.Context, id int, updated SyncSessionRecord) (ExistingSyncSessionRecord, error)
	DeleteSyncSessions(ctx context.Context, ids []any) ([]ExistingSyncSessionRecord, error)
}

// SyncSessionRecord represents the data needed to create a new sync session
type SyncSessionRecord struct {
	ServerURL string
	Status    SyncStatus
	Error     string
}

// ExistingSyncSessionRecord represents a sync session that exists in the database
type ExistingSyncSessionRecord struct {
	SyncSessionRecord
	ID        int
	CreatedAt time.Time
}

type SyncStatus int

const (
	_                 SyncStatus = iota
	SyncStatusPartial            // received 2xx response from sync server
	SyncStatusSuccess            // synced client db
	SyncStatusError
)
