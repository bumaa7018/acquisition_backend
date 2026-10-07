ALTER TABLE public.parcel
    DROP COLUMN IF EXISTS compensation_paid_full_at,
    DROP COLUMN IF EXISTS compensation_paid_40_at,
    DROP COLUMN IF EXISTS compensation_paid_60_at,
    DROP COLUMN IF EXISTS compensation_paid_full,
    DROP COLUMN IF EXISTS compensation_paid_40,
    DROP COLUMN IF EXISTS compensation_paid_60;
ALTER TABLE public.document DROP CONSTRAINT IF EXISTS document_payment_stage_check;
ALTER TABLE public.document DROP COLUMN IF EXISTS payment_date;
ALTER TABLE public.document DROP COLUMN IF EXISTS payment_stage;
