-- Захирамжийн төслийн санхүүжилт ОНООР: нэг эх үүсвэр он бүрт өөр дүнтэй
-- байж болно. Он нь холбоос (decision_draft_funding_source) дээр хадгалагдана.
--
-- Хуучин мөрийн онг seeder («000065_decision_funding_year») бөглөнө —
-- migration зөвхөн schema.
ALTER TABLE public.decision_draft_funding_source
    ADD COLUMN IF NOT EXISTS year integer;

-- Давхардлын дүрэм: (төсөл, эх үүсвэр) → (төсөл, эх үүсвэр, он).
-- NULLS NOT DISTINCT — онгүй хуучин мөр ч давхардахгүй (seeder бөглөхөөс өмнө).
ALTER TABLE public.decision_draft_funding_source
    DROP CONSTRAINT IF EXISTS decision_draft_funding_source_decision_draft_id_funding_sou_key;
ALTER TABLE public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_draft_source_year_key
    UNIQUE NULLS NOT DISTINCT (decision_draft_id, funding_source_id, year);

CREATE INDEX IF NOT EXISTS idx_ddfs_year
    ON public.decision_draft_funding_source USING btree (year);
