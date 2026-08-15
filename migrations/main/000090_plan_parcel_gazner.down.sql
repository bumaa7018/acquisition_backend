DROP INDEX IF EXISTS idx_plan_parcel_id;

ALTER TABLE plan
    DROP COLUMN IF EXISTS parcel_id,
    DROP COLUMN IF EXISTS gazner;
