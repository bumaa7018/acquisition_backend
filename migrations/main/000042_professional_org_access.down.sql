DROP INDEX IF EXISTS idx_la_professional_org;
DROP INDEX IF EXISTS idx_parcel_independent_org;
ALTER TABLE land_acquisition DROP COLUMN IF EXISTS professional_org_id;
ALTER TABLE parcel DROP COLUMN IF EXISTS independent_org_id;
