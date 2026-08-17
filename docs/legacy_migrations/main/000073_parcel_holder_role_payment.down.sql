DROP INDEX IF EXISTS idx_parcel_holder_role;
DROP INDEX IF EXISTS uq_parcel_holder_payment_recipient;

ALTER TABLE parcel_holder
    DROP CONSTRAINT IF EXISTS parcel_holder_holder_role_check;

ALTER TABLE parcel_holder
    DROP COLUMN IF EXISTS note,
    DROP COLUMN IF EXISTS payment_recipient,
    DROP COLUMN IF EXISTS holder_role;
