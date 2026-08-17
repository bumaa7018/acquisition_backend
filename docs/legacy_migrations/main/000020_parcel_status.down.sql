DROP INDEX IF EXISTS idx_parcel_status;

ALTER TABLE parcel
    DROP COLUMN IF EXISTS status;

DROP TABLE IF EXISTS parcel_status;
