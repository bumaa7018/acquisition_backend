ALTER TABLE parcel_valuation_submission DROP CONSTRAINT IF EXISTS pvs_acq_parcel_type_key;
ALTER TABLE parcel_valuation_submission ADD CONSTRAINT parcel_valuation_submission_acquisition_id_parcel_id_key
    UNIQUE (acquisition_id, parcel_id);

ALTER TABLE land_valuation DROP CONSTRAINT IF EXISTS land_valuation_acq_parcel_type_key;
ALTER TABLE land_valuation ADD CONSTRAINT land_valuation_acquisition_id_parcel_id_key
    UNIQUE (acquisition_id, parcel_id);

DROP INDEX IF EXISTS idx_compensation_valuation_type;
DROP INDEX IF EXISTS idx_assets_valuation_type;

ALTER TABLE parcel                              DROP COLUMN IF EXISTS selected_valuation_type;
ALTER TABLE parcel_valuation_submission_history DROP COLUMN IF EXISTS valuation_type;
ALTER TABLE parcel_valuation_submission         DROP COLUMN IF EXISTS valuation_type;
ALTER TABLE land_valuation                      DROP COLUMN IF EXISTS valuation_type;
ALTER TABLE compensation                        DROP COLUMN IF EXISTS valuation_type;
ALTER TABLE assets                              DROP COLUMN IF EXISTS valuation_type;
