CREATE TABLE parcel_workflow (
    id             SERIAL PRIMARY KEY,
    from_status_id INT REFERENCES parcel_status(id),
    to_status_id   INT NOT NULL REFERENCES parcel_status(id),
    sort_order     INT NOT NULL DEFAULT 0,
    UNIQUE(from_status_id, to_status_id)
);
