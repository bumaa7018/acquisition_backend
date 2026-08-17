CREATE TABLE acquisition_category (
    id        SERIAL PRIMARY KEY,
    name      VARCHAR(200) NOT NULL,
    parent_id INT REFERENCES acquisition_category(id),
    sort_order INT DEFAULT 0
);

ALTER TABLE land_acquisition
    ADD COLUMN general_category_id INT REFERENCES acquisition_category(id),
    ADD COLUMN sub_category_id     INT REFERENCES acquisition_category(id);
