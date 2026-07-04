BEGIN;

ALTER TABLE divisions
    ADD COLUMN IF NOT EXISTS discord_role_id VARCHAR(32) NULL,
    ADD COLUMN IF NOT EXISTS discord_announce_channel_id VARCHAR(32) NULL;

COMMIT;
