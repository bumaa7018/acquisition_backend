ALTER TABLE land_acquisition
    DROP COLUMN IF EXISTS project_name,
    DROP COLUMN IF EXISTS implementing_org,
    DROP COLUMN IF EXISTS reason,
    DROP COLUMN IF EXISTS responsible_org,
    DROP COLUMN IF EXISTS funding_source;
