CREATE TABLE IF NOT EXISTS sessions (
    id               TEXT    PRIMARY KEY,
    connector_type   TEXT    NOT NULL CHECK(connector_type IN ('whatsapp', 'discord', 'web')),
    channel_id       TEXT    NOT NULL,
    title            TEXT    NOT NULL DEFAULT '',
    system_prompt    TEXT    NOT NULL DEFAULT '',
    is_active        INTEGER NOT NULL DEFAULT 1,
    provider_override TEXT    NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(connector_type, channel_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id         TEXT    PRIMARY KEY,
    session_id TEXT    NOT NULL,
    role       TEXT    NOT NULL CHECK(role IN ('user', 'assistant', 'system')),
    content    TEXT    NOT NULL,
    timestamp  DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS config_entries (
    key        TEXT    PRIMARY KEY,
    value      TEXT    NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_connector_channel ON sessions(connector_type, channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_session_time ON messages(session_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS log_entries (
    id TEXT PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL CHECK(level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    logger TEXT NOT NULL,
    message TEXT NOT NULL,
    context TEXT,
    enabled BOOLEAN DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON log_entries(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_level ON log_entries(level);
CREATE INDEX IF NOT EXISTS idx_log_entries_logger ON log_entries(logger);

CREATE TABLE IF NOT EXISTS metric_snapshots (
    id TEXT PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    metric_name TEXT NOT NULL,
    metric_type TEXT NOT NULL CHECK(metric_type IN ('COUNTER', 'GAUGE', 'HISTOGRAM')),
    value REAL NOT NULL,
    labels TEXT,
    aggregation_window_seconds INTEGER DEFAULT 60
);
CREATE INDEX IF NOT EXISTS idx_metric_snapshots_timestamp ON metric_snapshots(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metric_snapshots_name ON metric_snapshots(metric_name);
CREATE INDEX IF NOT EXISTS idx_metric_snapshots_composite ON metric_snapshots(metric_name, timestamp DESC);

CREATE TABLE IF NOT EXISTS monitoring_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tool_executions (
    id          TEXT     PRIMARY KEY,
    session_id  TEXT     NOT NULL DEFAULT '',
    tool_name   TEXT     NOT NULL,
    input       TEXT     NOT NULL DEFAULT '{}',
    output      TEXT     NOT NULL DEFAULT '',
    latency_ms  INTEGER  NOT NULL DEFAULT 0,
    success     INTEGER  NOT NULL DEFAULT 1,
    error_msg   TEXT     NOT NULL DEFAULT '',
    executed_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS oauth_tokens (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id      TEXT     NOT NULL,
    provider     TEXT     NOT NULL,
    access_token TEXT     NOT NULL,
    refresh_token TEXT,
    expires_at   INTEGER,
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, provider)
);
