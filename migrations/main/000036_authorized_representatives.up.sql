CREATE TABLE IF NOT EXISTS authorized_representatives (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id UUID        NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    parcel_id      TEXT        NOT NULL DEFAULT '',
    last_name      TEXT        NOT NULL DEFAULT '',
    first_name     TEXT        NOT NULL DEFAULT '',
    register_no    TEXT        NOT NULL DEFAULT '',
    phone          TEXT        NOT NULL DEFAULT '',
    email          TEXT        NOT NULL DEFAULT '',
    address        TEXT        NOT NULL DEFAULT '',
    note           TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_authorized_representatives_acquisition_id ON authorized_representatives(acquisition_id);
CREATE INDEX IF NOT EXISTS idx_authorized_representatives_parcel_id      ON authorized_representatives(parcel_id);
