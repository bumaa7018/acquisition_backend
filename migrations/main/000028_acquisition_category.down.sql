ALTER TABLE land_acquisition
    DROP COLUMN IF EXISTS general_category_id,
    DROP COLUMN IF EXISTS sub_category_id;

DROP TABLE IF EXISTS acquisition_category;
