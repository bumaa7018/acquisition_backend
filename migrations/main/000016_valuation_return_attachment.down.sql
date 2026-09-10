ALTER TABLE public.parcel_valuation_submission
    DROP COLUMN IF EXISTS attachment_url,
    DROP COLUMN IF EXISTS attachment_name;

ALTER TABLE public.parcel_valuation_submission_history
    DROP COLUMN IF EXISTS attachment_url,
    DROP COLUMN IF EXISTS attachment_name;
