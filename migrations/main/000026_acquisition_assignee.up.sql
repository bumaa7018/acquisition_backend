CREATE TABLE land_acquisition_assignee (
    acquisition_id  UUID        NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL,
    user_name       TEXT        NOT NULL DEFAULT '',
    user_position   TEXT        NOT NULL DEFAULT '',
    assigned_by     UUID        NOT NULL,
    assigned_by_name TEXT       NOT NULL DEFAULT '',
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (acquisition_id, user_id)
);

CREATE INDEX idx_acq_assignee_acq_id  ON land_acquisition_assignee(acquisition_id);
CREATE INDEX idx_acq_assignee_user_id ON land_acquisition_assignee(user_id);
