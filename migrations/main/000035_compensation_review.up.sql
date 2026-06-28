ALTER TABLE compensation
    ADD COLUMN IF NOT EXISTS status       TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS review_note  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reviewed_by  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reviewed_at  TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS compensation_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    compensation_id UUID NOT NULL,
    acquisition_id  UUID NOT NULL,
    target_type     TEXT NOT NULL,
    parcel_id       TEXT NOT NULL DEFAULT '',
    asset_id        UUID,
    compensation_type TEXT NOT NULL,
    coverage_percent  NUMERIC(10,4) NOT NULL DEFAULT 0,
    amount          NUMERIC(20,2) NOT NULL DEFAULT 0,
    compensation_date DATE,
    note            TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'rejected',
    review_note     TEXT NOT NULL DEFAULT '',
    reviewed_by     TEXT NOT NULL DEFAULT '',
    reviewed_at     TIMESTAMPTZ,
    archived_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compensation_history_comp_id ON compensation_history(compensation_id);
