-- Нөхөх олговрын үнэлгээний ИЛГЭЭХ/ЗӨВШӨӨРӨХ төлөвийн флоу (нэгж талбар бүрт).
-- Нэгж талбарын статусын флоугаас ТУСДАА явна.
--   draft (Ноорог) → submitted (Илгээсэн) → approved (Баталгаажсан, эцсийн)
--                                          ↘ returned (Буцаагдсан) → дахин submitted
-- Илгээсэн төлөвт үнэлгээ засах боломжгүй; Баталгаажсан болсон үед л
-- нэгж талбар дараагийн явц руу шилжинэ.

CREATE TABLE IF NOT EXISTS parcel_valuation_submission (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    parcel_id      VARCHAR(12) NOT NULL REFERENCES parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'draft',   -- draft | submitted | approved | returned
    submitted_by   VARCHAR(255) NOT NULL DEFAULT '',
    submitted_at   TIMESTAMPTZ,
    reviewed_by    VARCHAR(255) NOT NULL DEFAULT '',
    reviewed_at    TIMESTAMPTZ,
    last_note      TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (acquisition_id, parcel_id)
);

CREATE INDEX IF NOT EXISTS idx_pvs_acq_parcel
    ON parcel_valuation_submission (acquisition_id, parcel_id);

-- Төлөв шилжилт бүрийн түүх (шилжилт тус бүрд нэг мөр, тайлбартай)
CREATE TABLE IF NOT EXISTS parcel_valuation_submission_history (
    id             SERIAL PRIMARY KEY,
    acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    parcel_id      VARCHAR(12) NOT NULL,
    action         TEXT NOT NULL,                    -- submit | approve | return
    from_status    TEXT NOT NULL DEFAULT '',
    to_status      TEXT NOT NULL,
    note           TEXT NOT NULL DEFAULT '',
    created_by     VARCHAR(255) NOT NULL DEFAULT 'system',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pvsh_acq_parcel
    ON parcel_valuation_submission_history (acquisition_id, parcel_id, created_at DESC);
