CREATE TABLE IF NOT EXISTS parcel_monitoring (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id          UUID        NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id     UUID        NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    monitoring_id      TEXT        NOT NULL UNIQUE,
    page_no            TEXT,
    status_name        TEXT,
    company_name       TEXT,
    source_created_at  TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE parcel_monitoring IS 'Нэгж талбарын хянан баталгааны мэдээлэл (middleware data_monitoring)';
COMMENT ON COLUMN parcel_monitoring.monitoring_id IS 'Эх системийн data_monitoring.mt_verify_requested.monitoring_id';
COMMENT ON COLUMN parcel_monitoring.source_created_at IS 'Эх системийн mt_verify_requested.created_at утга';

CREATE INDEX IF NOT EXISTS idx_parcel_monitoring_parcel ON parcel_monitoring (parcel_id);
CREATE INDEX IF NOT EXISTS idx_parcel_monitoring_acquisition ON parcel_monitoring (acquisition_id);
