ALTER TABLE compensation ADD COLUMN IF NOT EXISTS compensation_grant_id UUID REFERENCES compensation_grant(id) ON DELETE SET NULL;
UPDATE compensation c
SET compensation_grant_id = cg.id
FROM compensation_grant cg
WHERE cg.compensation_id = c.id;
ALTER TABLE compensation_grant DROP COLUMN IF EXISTS compensation_id;
