CREATE TABLE IF NOT EXISTS acquisition_workflow (
    id          SERIAL PRIMARY KEY,
    from_status INT,
    to_status   INT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_acquisition_workflow_from ON acquisition_workflow(from_status);
