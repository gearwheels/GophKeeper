CREATE TYPE secret_type AS ENUM (
    'login_password',
    'text',
    'binary',
    'card'
);

CREATE TABLE IF NOT EXISTS secrets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       secret_type NOT NULL,
    name       VARCHAR(255) NOT NULL,
    data       BYTEA NOT NULL,
    meta       TEXT,
    version    BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_secrets_updated_at ON secrets(updated_at);
