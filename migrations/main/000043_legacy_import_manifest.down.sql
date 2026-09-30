DROP TABLE IF EXISTS legacy_import_stage.rollback_log;

DELETE FROM legacy_import_stage.owned
 WHERE kind IN ('land_acquisition', 'parcel', 'valuation', 'funding_source', 'funding_link');

ALTER TABLE legacy_import_stage.owned DROP CONSTRAINT IF EXISTS legacy_import_owned_kind;
ALTER TABLE legacy_import_stage.owned ADD CONSTRAINT legacy_import_owned_kind
    CHECK (kind IN ('plan', 'right_holder', 'decision_draft'));

ALTER TABLE legacy_import_stage.owned DROP COLUMN IF EXISTS label;
