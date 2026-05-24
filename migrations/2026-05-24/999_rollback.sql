BEGIN;

DROP INDEX IF EXISTS idx_vms_vm_id;
DROP INDEX IF EXISTS idx_vms_user_challenge;
DROP INDEX IF EXISTS idx_vms_user_id;
DROP TABLE IF EXISTS vms;

ALTER TABLE challenges
    DROP COLUMN IF EXISTS vm_spec,
    DROP COLUMN IF EXISTS vm_enabled;

COMMIT;
