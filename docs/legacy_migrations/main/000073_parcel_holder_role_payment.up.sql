-- Итгэмжлэгдсэн төлөөлөгчийг эзэмшигчийн НЭГ ТӨРӨЛ болгож parcel_holder-т
-- нэгтгэнэ (тусдаа authorized_representatives хүснэгт хэрэглэхээ болино), мөн
-- нөхөн төлбөр хүлээн авах эзэмшигчийг сонгох баганыг нэмнэ.
--
-- Яагаад parcel_holder дээр: нэгж талбарын нөхөн төлбөрийг ХЭНД олгохыг
-- эзэмшигчдийн жагсаалтаас сонгодог тул сонголт нь эзэмшигчийн мөрийн шинж
-- чанар байх ёстой. Тусдаа хүснэгтэд байвал эзэмшигч устахад "өнчин" сонголт
-- үлдэнэ.

ALTER TABLE parcel_holder
    ADD COLUMN IF NOT EXISTS holder_role       VARCHAR(20) NOT NULL DEFAULT 'holder',
    ADD COLUMN IF NOT EXISTS payment_recipient BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS note              TEXT        NOT NULL DEFAULT '';

-- holder        — ГУС-аас татагдсан эзэмшигч (үндсэн болон хамтран)
-- representative— гараар бүртгэсэн итгэмжлэгдсэн төлөөлөгч
ALTER TABLE parcel_holder
    DROP CONSTRAINT IF EXISTS parcel_holder_holder_role_check;
ALTER TABLE parcel_holder
    ADD CONSTRAINT parcel_holder_holder_role_check
    CHECK (holder_role IN ('holder', 'representative'));

COMMENT ON COLUMN parcel_holder.holder_role       IS 'holder=ГУС-аас татагдсан эзэмшигч, representative=итгэмжлэгдсэн төлөөлөгч (гараар бүртгэнэ)';
COMMENT ON COLUMN parcel_holder.payment_recipient IS 'Нөхөн төлбөр хүлээн авах эзэмшигч — нэгж талбар тутамд ЗӨВХӨН НЭГ мөр TRUE байна';
COMMENT ON COLUMN parcel_holder.note              IS 'Итгэмжлэгдсэн төлөөлөгчийн тэмдэглэл';

-- Нэгж талбар тутамд төлбөр авагч ЗӨВХӨН нэг байхыг DB түвшинд баталгаажуулна.
CREATE UNIQUE INDEX IF NOT EXISTS uq_parcel_holder_payment_recipient
    ON parcel_holder (parcel_id)
    WHERE payment_recipient;

CREATE INDEX IF NOT EXISTS idx_parcel_holder_role
    ON parcel_holder (parcel_id, holder_role);
