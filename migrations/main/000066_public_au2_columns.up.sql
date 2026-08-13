ALTER TABLE public.au2
    DROP CONSTRAINT IF EXISTS au2_au1_code_fkey;

ALTER TABLE public.au2
    ADD COLUMN IF NOT EXISTS name_en VARCHAR(100);

ALTER TABLE public.au2
    ALTER COLUMN code TYPE VARCHAR(5) USING code::VARCHAR(5),
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name TYPE VARCHAR(50) USING name::VARCHAR(50),
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN area_m2 TYPE NUMERIC USING area_m2::NUMERIC,
    ALTER COLUMN geometry TYPE public.geometry(Polygon, 4326)
        USING CASE
            WHEN geometry IS NULL THEN NULL
            ELSE ST_GeometryN(ST_CollectionExtract(geometry, 3), 1)::public.geometry(Polygon, 4326)
        END,
    ALTER COLUMN au1_code TYPE VARCHAR(3) USING au1_code::VARCHAR(3),
    ALTER COLUMN au1_code DROP NOT NULL,
    ALTER COLUMN name_en TYPE VARCHAR(100) USING name_en::VARCHAR(100);

ALTER TABLE public.au2
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
     WHERE conrelid = 'public.au2'::regclass
       AND contype = 'p'
     LIMIT 1;

    IF existing_pk IS NOT NULL AND existing_pk <> 'au_level2_pkey' THEN
        EXECUTE format('ALTER TABLE public.au2 DROP CONSTRAINT %I', existing_pk);
    END IF;

    IF EXISTS (
        SELECT 1
          FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'au2'
           AND column_name = 'id'
    ) THEN
        ALTER TABLE public.au2 DROP COLUMN id;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au2'::regclass
           AND conname = 'au_level2_pkey'
    ) THEN
        ALTER TABLE public.au2
            ADD CONSTRAINT au_level2_pkey PRIMARY KEY (code);
    END IF;
END $$;

DROP INDEX IF EXISTS public.idx_au2_geometry;
DROP INDEX IF EXISTS public.idx_au2_au1_code;

CREATE UNIQUE INDEX IF NOT EXISTS au_level2_code_idx
    ON public.au2 USING btree (code);

CREATE INDEX IF NOT EXISTS st_idx_au_level2
    ON public.au2 USING gist (geometry);
