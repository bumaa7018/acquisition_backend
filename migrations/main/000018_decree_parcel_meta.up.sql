ALTER TABLE land_acquisition
    ADD COLUMN IF NOT EXISTS decree_number VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS decree_date   DATE;

ALTER TABLE parcel
    ADD COLUMN IF NOT EXISTS remaining_area_m2  NUMERIC(15,2),
    ADD COLUMN IF NOT EXISTS remaining_geom     geometry(Polygon, 4326),
    ADD COLUMN IF NOT EXISTS db_changed         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS changed_parcel_id  VARCHAR(50);
