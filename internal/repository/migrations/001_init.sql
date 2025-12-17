-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    initial TEXT NOT NULL,     -- JSON stored as TEXT
    response TEXT NOT NULL,    -- JSON stored as TEXT
    result TEXT NOT NULL       -- JSON stored as TEXT
);

CREATE TABLE IF NOT EXISTS nsx_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    host TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT,
    insecure INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_nsx_configs_name ON nsx_configs(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_nsx_configs_name;
DROP INDEX IF EXISTS idx_history_created_at;
DROP TABLE IF EXISTS nsx_configs;
DROP TABLE IF EXISTS history;
-- +goose StatementEnd
