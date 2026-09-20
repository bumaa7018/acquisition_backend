ALTER TABLE public.parcel_valuation_snapshot
    DROP COLUMN IF EXISTS source_name,
    DROP COLUMN IF EXISTS source_url;
