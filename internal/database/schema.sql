CREATE TABLE IF NOT EXISTS sessions (
    id               TEXT    PRIMARY KEY,
    connector_type   TEXT    NOT NULL CHECK(connector_type IN ('whatsapp', 'discord', 'telegram', 'web')),
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

CREATE TABLE IF NOT EXISTS proactive_tasks (
    id                 TEXT PRIMARY KEY,
    session_id         TEXT NOT NULL,
    connector_type     TEXT NOT NULL CHECK(connector_type IN ('whatsapp', 'discord', 'telegram', 'web')),
    channel_id         TEXT NOT NULL,
    target_connector   TEXT NOT NULL CHECK(target_connector IN ('whatsapp', 'discord', 'telegram', 'web')),
    target_channel_id  TEXT NOT NULL,
    title              TEXT NOT NULL,
    task_type          TEXT NOT NULL CHECK(task_type IN ('watch', 'cron')),
    schedule_expr      TEXT NOT NULL,
    timezone           TEXT NOT NULL DEFAULT 'UTC',
    prompt_condition   TEXT NOT NULL,
    target_tools       TEXT NOT NULL DEFAULT '[]',
    is_active          INTEGER NOT NULL DEFAULT 1,
    last_run_at        DATETIME,
    next_run_at        DATETIME NOT NULL,
    last_result_hash   TEXT NOT NULL DEFAULT '',
    created_at         DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at         DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_proactive_tasks_due ON proactive_tasks (is_active, next_run_at);
CREATE INDEX IF NOT EXISTS idx_proactive_tasks_session ON proactive_tasks (session_id);
