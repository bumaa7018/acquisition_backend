ALTER TABLE legacy_import_stage.report_row
    DROP COLUMN IF EXISTS id_kind,
    DROP COLUMN IF EXISTS assigned_id,
    DROP COLUMN IF EXISTS project;

ALTER TABLE legacy_import_stage.session DROP CONSTRAINT IF EXISTS legacy_import_session_mode;
ALTER TABLE legacy_import_stage.session DROP COLUMN IF EXISTS mode;
