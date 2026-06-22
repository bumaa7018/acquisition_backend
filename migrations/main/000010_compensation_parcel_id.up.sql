-- Add compensation_id to compensation_grant BEFORE dropping it from compensation
ALTER TABLE compensation_grant ADD COLUMN IF NOT EXISTS compensation_id UUID REFERENCES compensation(id) ON DELETE CASCADE;

-- Backfill: set compensation_grant.compensation_id from existing compensation.compensation_grant_id
UPDATE compensation_grant cg
SET compensation_id = c.id
FROM compensation c
WHERE c.compensation_grant_id = cg.id;

-- Drop compensation_grant_id from compensation (moved to grant side)
ALTER TABLE compensation DROP COLUMN IF EXISTS compensation_grant_id;
