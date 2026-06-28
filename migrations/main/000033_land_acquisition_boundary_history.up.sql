CREATE TABLE IF NOT EXISTS land_acquisition_boundary_history (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    land_acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    old_geometry        geometry(Polygon, 4326),
    new_geometry        geometry(Polygon, 4326) NOT NULL,
    changed_by          VARCHAR(255),
    changed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_acq
    ON land_acquisition_boundary_history (land_acquisition_id, changed_at DESC);

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_old_geom
    ON land_acquisition_boundary_history USING GIST (old_geometry);

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_new_geom
    ON land_acquisition_boundary_history USING GIST (new_geometry);
