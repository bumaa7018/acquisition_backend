DROP INDEX IF EXISTS public.idx_ddfs_year;

ALTER TABLE public.decision_draft_funding_source
    DROP CONSTRAINT IF EXISTS decision_draft_funding_source_draft_source_year_key;

-- Нэг эх үүсвэр олон онтой бол хуучин дүрэм (төсөл, эх үүсвэр) зөрчигдөнө —
-- тийм мөрийг гараар нэгтгэсний дараа буцаана.
ALTER TABLE public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_decision_draft_id_funding_sou_key
    UNIQUE (decision_draft_id, funding_source_id);

ALTER TABLE public.decision_draft_funding_source
    DROP COLUMN IF EXISTS year;
