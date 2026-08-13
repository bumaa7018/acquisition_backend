ALTER TABLE parcel_detail
    DROP COLUMN IF EXISTS record_status,
    DROP COLUMN IF EXISTS record_certificate_no,
    DROP COLUMN IF EXISTS record_date,
    DROP COLUMN IF EXISTS record_no,
    DROP COLUMN IF EXISTS contract_status,
    DROP COLUMN IF EXISTS contract_property_no,
    DROP COLUMN IF EXISTS contract_end,
    DROP COLUMN IF EXISTS contract_begin,
    DROP COLUMN IF EXISTS app_status,
    DROP COLUMN IF EXISTS app_type;

ALTER TABLE parcel
    DROP COLUMN IF EXISTS property_no,
    DROP COLUMN IF EXISTS address_streetname,
    DROP COLUMN IF EXISTS address_khashaa,
    DROP COLUMN IF EXISTS landuse_name;
