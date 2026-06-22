CREATE TABLE IF NOT EXISTS construction_type (
    id         SERIAL PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(200) NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0
);

INSERT INTO construction_type (code, name, sort_order) VALUES
    ('road',         'Зам, дэд бүтэц',              1),
    ('residential',  'Орон сууцны барилга',          2),
    ('industrial',   'Аж үйлдвэрийн байгууламж',    3),
    ('public',       'Нийтийн барилга байгууламж',  4),
    ('power_heat',   'Цахилгаан, дулааны шугам',    5),
    ('water',        'Усан хангамжийн байгууламж',   6),
    ('other',        'Бусад',                        7)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE land_acquisition
    ADD COLUMN IF NOT EXISTS construction_type_id INT REFERENCES construction_type(id) ON DELETE SET NULL;
