-- Захирамжийн төслийн санхүүгийн эх үүсвэрийг ОЛОН болгов.
--
-- Өмнө нь decision_draft.funding_source_id гэсэн нэг холбоос байсан бөгөөд
-- үүсгэх формд заавал бөглөгддөг байв. Одоо эх үүсвэрийг төслийг үүсгэсний
-- ДАРАА, дэлгэрэнгүй хэсгээс олноор нь нэмнэ.

CREATE TABLE IF NOT EXISTS decision_draft_funding_source (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_draft_id UUID NOT NULL REFERENCES decision_draft(id) ON DELETE CASCADE,
    funding_source_id UUID NOT NULL REFERENCES funding_sources(id) ON DELETE CASCADE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        TEXT        NOT NULL DEFAULT '',
    -- Нэг эх үүсвэрийг нэг захирамжид давхардуулж нэмэхгүй
    UNIQUE (decision_draft_id, funding_source_id)
);

CREATE INDEX IF NOT EXISTS idx_ddfs_draft  ON decision_draft_funding_source(decision_draft_id);
CREATE INDEX IF NOT EXISTS idx_ddfs_source ON decision_draft_funding_source(funding_source_id);

-- Хуучин нэг холбоосыг шинэ хүснэгт рүү зөөнө. Энэ нь "migration зөвхөн
-- schema" дүрмийн санаатай үл хамаарах тохиолдол: багана нь ЯГ ЭНЭ
-- migration-д устгагдаж байгаа тул зөөлтийг дараагийн seeder алхамд
-- үлдээвэл өгөгдөл эргэлт буцалтгүй алдагдана.
INSERT INTO decision_draft_funding_source (decision_draft_id, funding_source_id)
SELECT id, funding_source_id FROM decision_draft WHERE funding_source_id IS NOT NULL
ON CONFLICT DO NOTHING;

ALTER TABLE decision_draft DROP COLUMN IF EXISTS funding_source_id;
