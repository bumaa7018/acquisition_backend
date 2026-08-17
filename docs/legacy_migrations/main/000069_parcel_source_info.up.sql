ALTER TABLE parcel
    ADD COLUMN IF NOT EXISTS landuse_name TEXT,
    ADD COLUMN IF NOT EXISTS address_khashaa TEXT,
    ADD COLUMN IF NOT EXISTS address_streetname TEXT,
    ADD COLUMN IF NOT EXISTS property_no VARCHAR(100);

ALTER TABLE parcel_detail
    ADD COLUMN IF NOT EXISTS app_type TEXT,
    ADD COLUMN IF NOT EXISTS app_status TEXT,
    ADD COLUMN IF NOT EXISTS contract_begin DATE,
    ADD COLUMN IF NOT EXISTS contract_end DATE,
    ADD COLUMN IF NOT EXISTS contract_property_no VARCHAR(100),
    ADD COLUMN IF NOT EXISTS contract_status TEXT,
    ADD COLUMN IF NOT EXISTS record_no VARCHAR(100),
    ADD COLUMN IF NOT EXISTS record_date DATE,
    ADD COLUMN IF NOT EXISTS record_certificate_no VARCHAR(100),
    ADD COLUMN IF NOT EXISTS record_status TEXT;
