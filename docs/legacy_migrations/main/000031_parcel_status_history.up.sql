CREATE TABLE parcel_status_history (
    id             SERIAL PRIMARY KEY,
    parcel_id      VARCHAR(12)  NOT NULL REFERENCES parcel(parcel_id) ON DELETE CASCADE,
    acquisition_id UUID         NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    status_id      INT          NOT NULL REFERENCES parcel_status(id),
    status_date    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by     VARCHAR(255) NOT NULL DEFAULT 'system'
);

CREATE INDEX idx_psh_parcel      ON parcel_status_history(parcel_id);
CREATE INDEX idx_psh_acquisition ON parcel_status_history(acquisition_id);
