DROP INDEX IF EXISTS public.st_idx_au_level1;
DROP INDEX IF EXISTS public.au_level1_code_idx;

ALTER TABLE public.au1
    ADD COLUMN IF NOT EXISTS id UUID DEFAULT uuid_generate_v4(),
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_by VARCHAR(255);

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

    IF existing_pk IS NOT NULL AND existing_pk <> 'au1_pkey' THEN
        EXECUTE format('ALTER TABLE public.au1 DROP CONSTRAINT %I', existing_pk);
    END IF;

    UPDATE public.au1 SET id = uuid_generate_v4() WHERE id IS NULL;
    ALTER TABLE public.au1 ALTER COLUMN id SET NOT NULL;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au1'::regclass
           AND conname = 'au1_pkey'
    ) THEN
        ALTER TABLE public.au1
            ADD CONSTRAINT au1_pkey PRIMARY KEY (id);
    END IF;
END $$;

ALTER TABLE public.au1
    ALTER COLUMN code TYPE VARCHAR(10) USING code::VARCHAR(10),
    ALTER COLUMN name TYPE VARCHAR(100) USING name::VARCHAR(100),
    ALTER COLUMN area_m2 TYPE NUMERIC(18, 4) USING area_m2::NUMERIC(18, 4),
    ALTER COLUMN geometry TYPE public.geometry(Polygon, 4326)
        USING CASE
            WHEN geometry IS NULL THEN NULL
            ELSE ST_GeometryN(ST_CollectionExtract(geometry, 3), 1)::public.geometry(Polygon, 4326)
        END;

ALTER TABLE public.au1
    DROP COLUMN IF EXISTS name_en;

CREATE INDEX IF NOT EXISTS idx_au1_geometry ON public.au1 USING gist (geometry);
