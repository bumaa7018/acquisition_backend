-- Газрын төлбөрийн нэхэмжлэл — ГУС-аас татагдаж хадгалагдана.
-- invoice_no нь эх системийн давтагдашгүй дугаар тул давхардлыг үүгээр таьна.
CREATE TABLE IF NOT EXISTS parcel_invoice (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id      UUID        NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    invoice_no     TEXT        NOT NULL,
    description    TEXT,
    amount         NUMERIC(20,2),
    paid_amount    NUMERIC(20,2),
    status_id      INTEGER,
    status_name    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (parcel_id, invoice_no)
);

COMMENT ON TABLE parcel_invoice IS 'Нэгж талбарын газрын төлбөрийн нэхэмжлэл (ГУС data_estimate)';

CREATE INDEX IF NOT EXISTS idx_parcel_invoice_parcel ON parcel_invoice (parcel_id);
