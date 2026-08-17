-- Барьцаа болон шүүхийн шийдвэрийн бүртгэл — ГУС-аас татагдаж хадгалагдана.
-- Хоёулаа нэгж талбарын app_no-гоор татагддаг тул parcel-тай холбогдоно.

CREATE TABLE IF NOT EXISTS parcel_mortgage (
    id                    UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id             UUID        NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id        UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    -- source_id / source_parent_id нь ГУС-ийн ct_app8_ext.id / parent_id.
    -- Давхар барьцааг эцэгтэй нь холбоход хэрэглэнэ (эх системийн ID-гаар).
    source_id             TEXT        NOT NULL,
    source_parent_id      TEXT,
    app_no                TEXT,
    start_mortgage_period TEXT,
    end_mortgage_period   TEXT,
    loan_contract_no      TEXT,
    mortgage_contract_no  TEXT,
    loan_contract_amount  NUMERIC(20,2),
    monetary_unit_value   NUMERIC(20,2),
    unit_type             TEXT,
    mortgage_type         TEXT,
    status_id             TEXT,
    status_name           TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (parcel_id, source_id)
);

COMMENT ON TABLE  parcel_mortgage                  IS 'Нэгж талбарын барьцааны бүртгэл (ГУС ct_app8_ext)';
COMMENT ON COLUMN parcel_mortgage.source_parent_id IS 'Утгатай бол ДАВХАР барьцаа — эцэг барьцааны source_id';

CREATE INDEX IF NOT EXISTS idx_parcel_mortgage_parcel ON parcel_mortgage (parcel_id);
CREATE INDEX IF NOT EXISTS idx_parcel_mortgage_parent ON parcel_mortgage (parcel_id, source_parent_id);

CREATE TABLE IF NOT EXISTS parcel_court_decision (
    id                UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id         UUID        NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id    UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    source_id         TEXT        NOT NULL,
    app_no            TEXT,
    start_period      TEXT,
    end_period        TEXT,
    court_status      TEXT,
    status_name       TEXT,
    court_decision_no TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (parcel_id, source_id)
);

COMMENT ON TABLE parcel_court_decision IS 'Нэгж талбарт хамаарах шүүхийн шийдвэр (ГУС ct_app29_ext)';

CREATE INDEX IF NOT EXISTS idx_parcel_court_decision_parcel ON parcel_court_decision (parcel_id);
