ALTER TABLE legacy_import_stage.session DROP CONSTRAINT IF EXISTS legacy_import_session_status;
ALTER TABLE legacy_import_stage.session ADD CONSTRAINT legacy_import_session_status
    CHECK (status IN ('draft', 'committing', 'committed', 'failed'));
ALTER TABLE legacy_import_stage.session DROP COLUMN IF EXISTS rollback;
DROP TABLE IF EXISTS legacy_import_stage.owned;
DROP INDEX IF EXISTS land_acquisition_legacy_import_id;
ALTER TABLE public.land_acquisition DROP COLUMN IF EXISTS legacy_import_id;
