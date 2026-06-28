CREATE TABLE IF NOT EXISTS funding_sources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    organization_name TEXT NOT NULL,
    source_type   TEXT NOT NULL,
    amount        NUMERIC(20,2),
    currency      TEXT NOT NULL DEFAULT 'MNT',
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_funding_sources_acquisition_id ON funding_sources(acquisition_id);
