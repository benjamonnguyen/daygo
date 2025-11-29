package daygo

import (
	"context"
	"time"
)

type SyncSessionRepo interface {
	GetSession(ctx context.Context, id int) (ExistingSyncSessionRecord, error)
	GetLastSession(ctx context.Context, serverURL string, status SyncStatus) (ExistingSyncSessionRecord, error)
	InsertSession(ctx context.Context, session SyncSessionRecord) (ExistingSyncSessionRecord, error)
	UpdateSession(ctx context.Context, id int, updated SyncSessionRecord) (ExistingSyncSessionRecord, error)
	DeleteSession(ctx context.Context, id int) (ExistingSyncSessionRecord, error)
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

type SyncRequest struct {
	LastSyncTime time.Time            `json:"last_sync_time"`
	ClientTasks  []ExistingTaskRecord `json:"client_tasks"`
}

type SyncResponse struct {
	ServerTasks []ExistingTaskRecord `json:"server_tasks"`
}
