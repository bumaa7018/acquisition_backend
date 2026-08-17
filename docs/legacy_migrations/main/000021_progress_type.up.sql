CREATE TABLE IF NOT EXISTS progress_type (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    sort_order  INT
);
