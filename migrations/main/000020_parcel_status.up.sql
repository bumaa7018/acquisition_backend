CREATE TABLE IF NOT EXISTS parcel_status (
    id         INT          PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(200) NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0
);

INSERT INTO parcel_status (id, code, name, sort_order) VALUES
    (0, 'pending',      'Хүлээгдэж буй',            0),
    (1, 'processing',      'Зөвшилцөх шатандаа',    1),
    (2, 'evaluating',      'Үнэлгээ хийх',          2),
    (3, 'affected_out',  'Нөлөөлөгдсөн гарсан',     3),
    (4, 'rejected',      'Татгалзсан',              4),
    (5, 'released',   'Чөлөөлсөн',                  5)
ON CONFLICT (id) DO UPDATE SET
    code = EXCLUDED.code,
    name = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order;

ALTER TABLE parcel
    ADD COLUMN IF NOT EXISTS status INT NOT NULL DEFAULT 5 REFERENCES parcel_status(id);

ALTER TABLE parcel
    ALTER COLUMN status SET DEFAULT 5;

CREATE INDEX IF NOT EXISTS idx_parcel_status ON parcel (status);
