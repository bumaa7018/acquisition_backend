-- Захирамжийн төсөл (decision_draft) — саналын хуудаснаас захирамж болтол
-- явагдах бүртгэл. Нэгж талбарууд нь decision_draft_parcel-ээр холбогдоно.
--
-- Ажлын төрөл / Төсөв нь Тохиргоо цэснээс удирдагдах сонголтын жагсаалт тул
-- тусдаа lookup хүснэгт болгов. Анхны утгуудыг ЭНД оруулахгүй — seeder-ээр
-- (internal/seed) оруулна (migration зөвхөн schema).

CREATE TABLE IF NOT EXISTS decision_work_type (
    id         SERIAL       PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(200) NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS decision_budget (
    id         SERIAL       PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(200) NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS decision_draft (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Саналын хуудасны дугаар — үүсгэх үед заавал
    proposal_no       TEXT NOT NULL,
    -- Захирамжийн дугаар/огноо — "Явц нэмэх" үед бөглөгдөж баталгаажина
    decree_number     VARCHAR(100) NOT NULL DEFAULT '',
    decision_date     DATE,
    location          TEXT NOT NULL DEFAULT '',
    -- Хугацаа (он)
    duration_year     INT,
    -- 1 = Төсөл, 2 = Баталгаажсан
    status            SMALLINT NOT NULL DEFAULT 1,
    -- Эхний нэгж талбар холбогдоход тухайн талбарын чөлөөлөлтөөр бөглөгдөнө
    acquisition_id    UUID REFERENCES land_acquisition(id) ON DELETE SET NULL,
    funding_source_id UUID REFERENCES funding_sources(id) ON DELETE SET NULL,
    work_type_id      INT  REFERENCES decision_work_type(id) ON DELETE SET NULL,
    budget_id         INT  REFERENCES decision_budget(id) ON DELETE SET NULL,
    confirmed_at      TIMESTAMPTZ,
    confirmed_by      TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        TEXT        NOT NULL DEFAULT '',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by        TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_decision_draft_status        ON decision_draft(status);
CREATE INDEX IF NOT EXISTS idx_decision_draft_acquisition   ON decision_draft(acquisition_id);
CREATE INDEX IF NOT EXISTS idx_decision_draft_decree_number ON decision_draft(decree_number);
CREATE INDEX IF NOT EXISTS idx_decision_draft_proposal_no   ON decision_draft(proposal_no);

-- Холбоос хүснэгт. Хасалт нь мөр УСТГАХГҮЙ — removed_at тэмдэглэж түүх үлдээнэ.
CREATE TABLE IF NOT EXISTS decision_draft_parcel (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_draft_id UUID NOT NULL REFERENCES decision_draft(id) ON DELETE CASCADE,
    parcel_id         UUID NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    linked_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by         TEXT        NOT NULL DEFAULT '',
    removed_at        TIMESTAMPTZ,
    removed_by        TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ddp_draft  ON decision_draft_parcel(decision_draft_id);
CREATE INDEX IF NOT EXISTS idx_ddp_parcel ON decision_draft_parcel(parcel_id);

-- Нэг нэгж талбар зөвхөн НЭГ идэвхтэй захирамжтай холбогдоно. Хасагдсаны
-- (removed_at IS NOT NULL) дараа өөр захирамжид дахин холбогдож болно.
CREATE UNIQUE INDEX IF NOT EXISTS uq_ddp_active_parcel
    ON decision_draft_parcel(parcel_id) WHERE removed_at IS NULL;
