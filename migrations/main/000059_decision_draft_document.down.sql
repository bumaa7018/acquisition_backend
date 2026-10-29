DROP INDEX IF EXISTS public.idx_decision_draft_document;
ALTER TABLE public.decision_draft DROP COLUMN IF EXISTS document_id;
