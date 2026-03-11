CREATE TABLE IF NOT EXISTS sessions (
    id             TEXT    PRIMARY KEY,
    connector_type TEXT    NOT NULL CHECK(connector_type IN ('whatsapp', 'discord')),
    channel_id     TEXT    NOT NULL,
    system_prompt  TEXT    NOT NULL DEFAULT '',
    is_active      INTEGER NOT NULL DEFAULT 1,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at     DATETIME NOT NULL DEFAULT (datetime('now')),
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
