CREATE TABLE IF NOT EXISTS secrets (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    type TEXT NOT NULL,
    payload BYTEA NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_secrets_owner_id ON secrets(owner_id);
CREATE INDEX IF NOT EXISTS idx_secrets_owner_updated ON secrets(owner_id, updated_at);
