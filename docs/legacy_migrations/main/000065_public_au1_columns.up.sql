ALTER TABLE public.au1
    ADD COLUMN IF NOT EXISTS name_en VARCHAR(100);

ALTER TABLE public.au1
    ALTER COLUMN code TYPE VARCHAR(3) USING code::VARCHAR(3),
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name TYPE VARCHAR(50) USING name::VARCHAR(50),
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN area_m2 TYPE NUMERIC USING area_m2::NUMERIC,
    ALTER COLUMN geometry TYPE public.geometry(MultiPolygon, 4326)
        USING CASE
            WHEN geometry IS NULL THEN NULL
            ELSE ST_Multi(ST_CollectionExtract(geometry, 3))::public.geometry(MultiPolygon, 4326)
        END,
    ALTER COLUMN name_en TYPE VARCHAR(100) USING name_en::VARCHAR(100);

ALTER TABLE public.au1
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
     WHERE conrelid = 'public.au1'::regclass
       AND contype = 'p'
     LIMIT 1;

    IF existing_pk IS NOT NULL AND existing_pk <> 'au_level1_pkey' THEN
        EXECUTE format('ALTER TABLE public.au1 DROP CONSTRAINT %I', existing_pk);
    END IF;

    IF EXISTS (
        SELECT 1
          FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'au1'
           AND column_name = 'id'
    ) THEN
        ALTER TABLE public.au1 DROP COLUMN id;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au1'::regclass
           AND conname = 'au_level1_pkey'
    ) THEN
        ALTER TABLE public.au1
            ADD CONSTRAINT au_level1_pkey PRIMARY KEY (code);
    END IF;
END $$;

DROP INDEX IF EXISTS public.idx_au1_geometry;

CREATE UNIQUE INDEX IF NOT EXISTS au_level1_code_idx
    ON public.au1 USING btree (code);

CREATE INDEX IF NOT EXISTS st_idx_au_level1
    ON public.au1 USING gist (geometry);
