CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    name TEXT NOT NULL,
    started_at INTEGER,
    ended_at INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES tasks(id)
);
