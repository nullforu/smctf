BEGIN;

ALTER TABLE divisions
    DROP COLUMN IF EXISTS discord_role_id,
    DROP COLUMN IF EXISTS discord_announce_channel_id;

COMMIT;
