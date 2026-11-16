DROP INDEX IF EXISTS legacy_import_stage.legacy_import_session_owner;
ALTER TABLE legacy_import_stage.session DROP COLUMN IF EXISTS created_by_id;
