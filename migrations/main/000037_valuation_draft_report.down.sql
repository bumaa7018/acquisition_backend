ALTER TABLE public.parcel_valuation_submission
    DROP COLUMN IF EXISTS draft_report_name,
    DROP COLUMN IF EXISTS draft_report_url;
