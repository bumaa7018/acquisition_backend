DROP INDEX IF EXISTS public.idx_land_release_structure_type;
ALTER TABLE public.land_release DROP COLUMN IF EXISTS structure_type_id;
ALTER TABLE public.land_release_import DROP COLUMN IF EXISTS structure_type_id;
DROP TABLE IF EXISTS public.land_release_structure_type;
