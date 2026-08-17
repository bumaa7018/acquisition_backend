ALTER TABLE public.au3
    DROP CONSTRAINT IF EXISTS au3_au1_code_fkey,
    DROP CONSTRAINT IF EXISTS au3_au2_code_fkey;

ALTER TABLE public.au3
    ADD COLUMN IF NOT EXISTS name_desc TEXT,
    ADD COLUMN IF NOT EXISTS old_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS name_en VARCHAR(100);

ALTER TABLE public.au3
    ALTER COLUMN code TYPE VARCHAR(8) USING code::VARCHAR(8),
    ALTER COLUMN code DROP NOT NULL,
    ALTER COLUMN name TYPE VARCHAR(50) USING name::VARCHAR(50),
    ALTER COLUMN name DROP NOT NULL,
    ALTER COLUMN area_m2 TYPE NUMERIC USING area_m2::NUMERIC,
    ALTER COLUMN geometry TYPE public.geometry(MultiPolygon, 4326)
        USING CASE
            WHEN geometry IS NULL THEN NULL
            ELSE ST_Multi(ST_CollectionExtract(geometry, 3))::public.geometry(MultiPolygon, 4326)
        END,
    ALTER COLUMN au1_code TYPE VARCHAR(3) USING au1_code::VARCHAR(3),
    ALTER COLUMN au1_code DROP NOT NULL,
    ALTER COLUMN au2_code TYPE VARCHAR(5) USING au2_code::VARCHAR(5),
    ALTER COLUMN au2_code DROP NOT NULL,
    ALTER COLUMN name_en TYPE VARCHAR(100) USING name_en::VARCHAR(100);

ALTER TABLE public.au3
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS updated_by;

DO $$
DECLARE
    existing_pk TEXT;
BEGIN
    SELECT conname
      INTO existing_pk
      FROM pg_constraint
     WHERE conrelid = 'public.au3'::regclass
       AND contype = 'p'
     LIMIT 1;

    IF existing_pk IS NOT NULL THEN
        EXECUTE format('ALTER TABLE public.au3 DROP CONSTRAINT %I', existing_pk);
    END IF;

    IF EXISTS (
        SELECT 1
          FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'au3'
           AND column_name = 'id'
    ) THEN
        ALTER TABLE public.au3 DROP COLUMN id;
    END IF;
END $$;

DROP INDEX IF EXISTS public.idx_au3_geometry;
DROP INDEX IF EXISTS public.idx_au3_au1_code;
DROP INDEX IF EXISTS public.idx_au3_au2_code;

ALTER TABLE public.au1 DROP CONSTRAINT IF EXISTS au1_code_key;
ALTER TABLE public.au2 DROP CONSTRAINT IF EXISTS au2_code_key;
ALTER TABLE public.au3 DROP CONSTRAINT IF EXISTS au3_code_key;

CREATE UNIQUE INDEX IF NOT EXISTS au_level3_code_idx
    ON public.au3 USING btree (code);

CREATE INDEX IF NOT EXISTS st_idx_au_level3
    ON public.au3 USING gist (geometry);
