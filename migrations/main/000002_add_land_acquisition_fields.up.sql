ALTER TABLE land_acquisition
    ADD COLUMN IF NOT EXISTS project_name      TEXT,
    ADD COLUMN IF NOT EXISTS implementing_org  TEXT,
    ADD COLUMN IF NOT EXISTS reason            TEXT,
    ADD COLUMN IF NOT EXISTS responsible_org   TEXT,
    ADD COLUMN IF NOT EXISTS funding_source    TEXT;
