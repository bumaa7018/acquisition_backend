-- Хуучин мэдээллийн импортын БҮРТГЭЛ (manifest) ба УСТГАЛЫН ЛОГ.
--
-- owned — импорт ӨӨРӨӨ үүсгэсэн мөр бүр (чөлөөлөлт, нэгж талбар, үнэлгээ,
-- санхүүжилт, захирамж …). «Устгах» нь ЗӨВХӨН энэ бүртгэлийг харж мөр бүрийг
-- нэг бүрчлэн устгана; өөр газраас аль хэдийн устсан бол логт бичээд алгасна.
ALTER TABLE legacy_import_stage.owned ADD COLUMN IF NOT EXISTS label text NOT NULL DEFAULT '';

ALTER TABLE legacy_import_stage.owned DROP CONSTRAINT IF EXISTS legacy_import_owned_kind;
ALTER TABLE legacy_import_stage.owned ADD CONSTRAINT legacy_import_owned_kind
    CHECK (kind IN ('plan', 'right_holder', 'decision_draft', 'land_acquisition', 'parcel',
                    'valuation', 'funding_source', 'funding_link'));

-- rollback_log — «Устгах»-ын алхам бүрийн түүх (дэлгэцэд харуулна).
CREATE TABLE IF NOT EXISTS legacy_import_stage.rollback_log (
    id         bigserial PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES legacy_import_stage.session(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT '',
    step       text NOT NULL,
    kind       text NOT NULL,
    ref_id     text NOT NULL DEFAULT '',
    label      text NOT NULL DEFAULT '',
    action     text NOT NULL,
    message    text NOT NULL DEFAULT '',
    CONSTRAINT legacy_import_rollback_log_action
        CHECK (action IN ('deleted', 'detached', 'skipped', 'kept', 'failed'))
);
CREATE INDEX IF NOT EXISTS legacy_import_rollback_log_session
    ON legacy_import_stage.rollback_log (session_id, id);

-- Өмнөх хувилбараар (000042) оруулсан импортын бүртгэлийг нөхнө:
-- тэр үед импорт зөвхөн ӨӨРИЙН чөлөөлөлтөд created_by='legacy-import' мөр бичдэг байв.
INSERT INTO legacy_import_stage.owned (session_id, kind, ref_id, label)
SELECT la.legacy_import_id, 'land_acquisition', la.id, COALESCE(la.acquisition_name, '')
  FROM public.land_acquisition la
 WHERE la.legacy_import_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO legacy_import_stage.owned (session_id, kind, ref_id, label)
SELECT la.legacy_import_id, 'parcel', p.id, p.parcel_id
  FROM public.parcel p JOIN public.land_acquisition la ON la.id = p.acquisition_id
 WHERE la.legacy_import_id IS NOT NULL AND p.created_by = 'legacy-import'
ON CONFLICT DO NOTHING;

INSERT INTO legacy_import_stage.owned (session_id, kind, ref_id, label)
SELECT DISTINCT la.legacy_import_id, 'valuation', p.id, p.parcel_id
  FROM public.parcel p JOIN public.land_acquisition la ON la.id = p.acquisition_id
  JOIN public.compensation c ON c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
 WHERE la.legacy_import_id IS NOT NULL AND p.created_by = 'legacy-import' AND c.created_by = 'legacy-import'
ON CONFLICT DO NOTHING;

INSERT INTO legacy_import_stage.owned (session_id, kind, ref_id, label)
SELECT la.legacy_import_id, 'funding_source', f.id, COALESCE(f.organization_name, '')
  FROM public.funding_sources f JOIN public.land_acquisition la ON la.id = f.acquisition_id
 WHERE la.legacy_import_id IS NOT NULL AND f.created_by = 'legacy-import'
ON CONFLICT DO NOTHING;

INSERT INTO legacy_import_stage.owned (session_id, kind, ref_id, label)
SELECT la.legacy_import_id, 'funding_link', d.id, ''
  FROM public.decision_draft_funding_source d
  JOIN public.funding_sources f ON f.id = d.funding_source_id
  JOIN public.land_acquisition la ON la.id = f.acquisition_id
 WHERE la.legacy_import_id IS NOT NULL AND d.created_by = 'legacy-import'
ON CONFLICT DO NOTHING;
