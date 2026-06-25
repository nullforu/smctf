BEGIN;

CREATE TABLE IF NOT EXISTS discord_connections (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    discord_user_id VARCHAR(32) NOT NULL,
    discord_username VARCHAR(100) NULL,
    discord_global_name VARCHAR(100) NULL,
    discord_avatar VARCHAR(255) NULL,
    role_status VARCHAR(30) NOT NULL DEFAULT 'CONNECTED',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    last_synced_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_discord_connections_user ON discord_connections (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_discord_connections_discord_user ON discord_connections (discord_user_id);

COMMIT;
