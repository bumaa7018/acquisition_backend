ALTER TABLE public.parcel
    DROP COLUMN IF EXISTS estimated_value,
    DROP COLUMN IF EXISTS estimated_value_at,
    DROP COLUMN IF EXISTS estimated_value_by;
