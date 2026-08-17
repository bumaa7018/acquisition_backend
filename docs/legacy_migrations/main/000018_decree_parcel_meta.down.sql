ALTER TABLE land_acquisition
    DROP COLUMN IF EXISTS decree_number,
    DROP COLUMN IF EXISTS decree_date;

ALTER TABLE parcel
    DROP COLUMN IF EXISTS remaining_area_m2,
    DROP COLUMN IF EXISTS remaining_geom,
    DROP COLUMN IF EXISTS db_changed,
    DROP COLUMN IF EXISTS changed_parcel_id;
