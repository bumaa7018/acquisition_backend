CREATE TABLE IF NOT EXISTS compensation_grant (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id   UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    amount           NUMERIC(20,2) NOT NULL DEFAULT 0,
    grant_date       DATE,
    note             TEXT NOT NULL DEFAULT '',
    land_area_m2     NUMERIC(15,2) NOT NULL DEFAULT 0,
    land_price       NUMERIC(20,2) NOT NULL DEFAULT 0,
    land_location    TEXT NOT NULL DEFAULT '',
    land_purpose     TEXT NOT NULL DEFAULT '',
    land_use_type    TEXT NOT NULL DEFAULT '',
    parcel_number    VARCHAR(50) NOT NULL DEFAULT '',
    decree_number    VARCHAR(100) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       TEXT NOT NULL DEFAULT '',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compensation (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id       UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    asset_type           VARCHAR(20) NOT NULL DEFAULT 'property',
    parcel_code          VARCHAR(50) NOT NULL DEFAULT '',
    asset_id             UUID REFERENCES assets(id) ON DELETE SET NULL,
    compensation_type    VARCHAR(20) NOT NULL DEFAULT 'cash',
    coverage_percent     NUMERIC(5,2) NOT NULL DEFAULT 100,
    amount               NUMERIC(20,2) NOT NULL DEFAULT 0,
    compensation_date    DATE,
    note                 TEXT NOT NULL DEFAULT '',
    compensation_grant_id UUID REFERENCES compensation_grant(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by           TEXT NOT NULL DEFAULT '',
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE parcel DROP COLUMN IF EXISTS acquisition_amount;
ALTER TABLE parcel DROP COLUMN IF EXISTS compensation_types;
ALTER TABLE assets DROP COLUMN IF EXISTS acquisition_amount;
