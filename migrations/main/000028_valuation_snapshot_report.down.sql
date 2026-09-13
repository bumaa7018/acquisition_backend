ALTER TABLE public.parcel_valuation_snapshot
    DROP COLUMN IF EXISTS report_url,
    DROP COLUMN IF EXISTS report_name;
