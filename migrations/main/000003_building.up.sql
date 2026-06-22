CREATE TABLE building (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id  UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    parcel_id       VARCHAR(12) NOT NULL REFERENCES parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE,
    building_number TEXT             NOT NULL DEFAULT '',
    building_type   TEXT             NOT NULL DEFAULT '',
    floor_count     INT              NOT NULL DEFAULT 0,
    area_m2         NUMERIC(18, 4)   NOT NULL DEFAULT 0,
    acquisition_amount NUMERIC(18, 4) NOT NULL DEFAULT 0,
    owner_name      TEXT             NOT NULL DEFAULT '',
    address         TEXT             NOT NULL DEFAULT '',
    notes           TEXT             NOT NULL DEFAULT '',
    geometry        geometry(Polygon, 4326),
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_building_acquisition_id ON building(acquisition_id);
CREATE INDEX idx_building_parcel_id      ON building(parcel_id);
CREATE INDEX idx_building_geometry       ON building USING GIST (geometry);
