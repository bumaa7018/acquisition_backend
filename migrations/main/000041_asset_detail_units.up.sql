-- Drop the wide asset_detail table (replaced by normalized spec/calc tables)
DROP TABLE IF EXISTS asset_detail;

-- Building spec types lookup
CREATE TABLE IF NOT EXISTS asset_spec_type (
    id         SERIAL PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    sort_order INT  NOT NULL DEFAULT 0
);

INSERT INTO asset_spec_type (code, name, sort_order) VALUES
    ('construction_type', 'Хийцийн төрөл',        1),
    ('foundation',        'Суурь бүтэц',           2),
    ('walls',             'Хана',                  3),
    ('roof',              'Хучилт-дээвэр',         4),
    ('door_type',         'Хаалга',                5),
    ('window_type',       'Цонх',                  6),
    ('heating',           'Халаалтын шийдэл',      7),
    ('quality',           'Чанар',                 8)
ON CONFLICT (code) DO NOTHING;

-- Per-asset building spec values (text)
CREATE TABLE IF NOT EXISTS asset_spec (
    id           UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id     UUID    NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    spec_type_id INT     NOT NULL REFERENCES asset_spec_type(id),
    value        TEXT    NOT NULL DEFAULT '',
    UNIQUE(asset_id, spec_type_id)
);

-- Calculation type lookup
CREATE TABLE IF NOT EXISTS asset_calc_type (
    id           SERIAL PRIMARY KEY,
    code         TEXT   NOT NULL UNIQUE,
    name         TEXT   NOT NULL,
    default_unit TEXT   NOT NULL DEFAULT '',
    sort_order   INT    NOT NULL DEFAULT 0
);

INSERT INTO asset_calc_type (code, name, default_unit, sort_order) VALUES
    ('unit_cost',              'Барилгын нэгжийн өртөг',                  'төгрөг/м²', 1),
    ('coeff_price_increase',   'Үнийн өсөлт',                             '%',         2),
    ('coeff_interior',         'Гоёлын дотор засалтай',                   '%',         3),
    ('coeff_casting',          'Хийц болон давхар хоорондын цутгалт',     '%',         4),
    ('coeff_brick',            'Тоосгон өнгөлгөө',                        '%',         5),
    ('height',                 'Өндөр',                                   'м',         6),
    ('full_replacement_cost',  'Бүрэн орлуулах өртөг',                    '₮',         7),
    ('depreciation_percent',   'Элэгдлийн хувь',                         '%',         8),
    ('depreciation_amount',    'Элэгдлийн дүн',                           '₮',         9),
    ('net_replacement_cost',   'Барилгын нөхөн орлуулах өртөг',           '₮',         10)
ON CONFLICT (code) DO NOTHING;

-- Per-asset calculation values (numeric + user-entered unit)
CREATE TABLE IF NOT EXISTS asset_calculation (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id     UUID         NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    calc_type_id INT          NOT NULL REFERENCES asset_calc_type(id),
    unit         TEXT         NOT NULL DEFAULT '',
    value        NUMERIC(18,4) NOT NULL DEFAULT 0,
    UNIQUE(asset_id, calc_type_id)
);
