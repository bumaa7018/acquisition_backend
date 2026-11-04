DROP INDEX IF EXISTS public.idx_land_release_au2;
ALTER TABLE public.land_release
    DROP COLUMN IF EXISTS au2_code,
    DROP COLUMN IF EXISTS au1_code;
