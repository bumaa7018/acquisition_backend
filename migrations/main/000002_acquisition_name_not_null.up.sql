UPDATE land_acquisition SET acquisition_name = '' WHERE acquisition_name IS NULL;
ALTER TABLE land_acquisition ALTER COLUMN acquisition_name SET NOT NULL;
ALTER TABLE land_acquisition ALTER COLUMN acquisition_name SET DEFAULT '';
