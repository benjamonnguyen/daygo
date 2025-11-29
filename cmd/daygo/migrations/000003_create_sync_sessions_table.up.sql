CREATE TABLE IF NOT EXISTS sync_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_url TEXT NOT NULL,
    status INTEGER NOT NULL,
    error TEXT,
    created_at INTEGER NOT NULL
);
