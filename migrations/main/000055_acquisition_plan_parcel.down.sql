DROP INDEX IF EXISTS public.idx_land_acquisition_plan_parcel_id;

ALTER TABLE public.land_acquisition
    DROP COLUMN IF EXISTS plan_gazner,
    DROP COLUMN IF EXISTS plan_parcel_id;
