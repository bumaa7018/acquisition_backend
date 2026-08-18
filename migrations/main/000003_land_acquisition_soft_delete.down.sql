DROP INDEX IF EXISTS public.idx_land_acquisition_active;

ALTER TABLE public.land_acquisition
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by;

DROP INDEX IF EXISTS public.idx_parcel_active;

ALTER TABLE public.parcel
    DROP COLUMN IF EXISTS deleted_at;
