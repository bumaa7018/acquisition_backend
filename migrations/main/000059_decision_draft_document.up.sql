-- Захирамжийг БАТАЛГААЖУУЛАХАД заавал хавсаргах файл (захирамжийн эх баримт).
-- Файл нь document хүснэгтэд «Захирамж» (decree) төрлөөр бичигдэж, захирамжид
-- холбогдсон бүх нэгж талбарын хавсралт (parcel_document) болж харагдана.
-- Төрлийн мөрийг seeder нэмнэ (migration = зөвхөн бүтэц).
ALTER TABLE public.decision_draft
    ADD COLUMN IF NOT EXISTS document_id uuid
        REFERENCES public.document(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_decision_draft_document
    ON public.decision_draft (document_id)
    WHERE document_id IS NOT NULL;
