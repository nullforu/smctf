BEGIN;

ALTER TABLE challenges
    ADD COLUMN IF NOT EXISTS vm_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS vm_spec TEXT;

CREATE TABLE IF NOT EXISTS vms (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id BIGINT NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    vm_id TEXT NOT NULL,
    status TEXT NOT NULL,
    node_name TEXT,
    external_ip TEXT,
    ports JSONB,
    ttl_expires_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vms_user_id ON vms (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_vms_user_challenge ON vms (user_id, challenge_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_vms_vm_id ON vms (vm_id);

COMMIT;
