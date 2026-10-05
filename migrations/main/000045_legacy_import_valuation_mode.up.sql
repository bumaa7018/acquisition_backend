-- Хуучин мэдээллийн импортын ГОРИМ:
--   full      — чөлөөлөлтийн хил + нэгж талбарын SHP + үнэлгээний эксэл
--   valuation — ЗӨВХӨН үнэлгээний эксэл (архивын импорттой ижил: чөлөөлөлтийг
--               «Бүтээн байгуулалтын ажлын нэр»-ээр бүлэглэж ХИЛГҮЙ үүсгэнэ,
--               нэгж талбарыг ГУС-аас татна)
ALTER TABLE legacy_import_stage.session
    ADD COLUMN IF NOT EXISTS mode text NOT NULL DEFAULT 'full';
ALTER TABLE legacy_import_stage.session DROP CONSTRAINT IF EXISTS legacy_import_session_mode;
ALTER TABLE legacy_import_stage.session ADD CONSTRAINT legacy_import_session_mode
    CHECK (mode IN ('full', 'valuation'));

-- «Зөвхөн үнэлгээ» горимын мөр бүрийн шийдвэр:
--   project     — нэгтгэсэн төслийн нэр (чөлөөлөлт)
--   assigned_id — системд бүртгэгдэх дугаар (бодит, D<n>… эсвэл ERR…)
--   id_kind     — real | duplicate | synthetic
ALTER TABLE legacy_import_stage.report_row
    ADD COLUMN IF NOT EXISTS project text,
    ADD COLUMN IF NOT EXISTS assigned_id text,
    ADD COLUMN IF NOT EXISTS id_kind text;
