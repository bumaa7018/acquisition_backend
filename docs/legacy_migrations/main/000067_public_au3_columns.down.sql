DROP INDEX IF EXISTS public.st_idx_au_level3;
DROP INDEX IF EXISTS public.au_level3_code_idx;

ALTER TABLE public.au3
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
     WHERE conrelid = 'public.au3'::regclass
       AND contype = 'p'
     LIMIT 1;

    IF existing_pk IS NOT NULL AND existing_pk <> 'au3_pkey' THEN
        EXECUTE format('ALTER TABLE public.au3 DROP CONSTRAINT %I', existing_pk);
    END IF;

    UPDATE public.au3 SET id = uuid_generate_v4() WHERE id IS NULL;
    ALTER TABLE public.au3 ALTER COLUMN id SET NOT NULL;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au3'::regclass
           AND conname = 'au3_pkey'
    ) THEN
        ALTER TABLE public.au3
            ADD CONSTRAINT au3_pkey PRIMARY KEY (id);
    END IF;
END $$;

ALTER TABLE public.au3
    ALTER COLUMN code TYPE VARCHAR(10) USING code::VARCHAR(10),
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name TYPE VARCHAR(100) USING name::VARCHAR(100),
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN au1_code TYPE VARCHAR(10) USING au1_code::VARCHAR(10),
    ALTER COLUMN au1_code SET NOT NULL,
    ALTER COLUMN au2_code TYPE VARCHAR(10) USING au2_code::VARCHAR(10),
    ALTER COLUMN au2_code SET NOT NULL,
    ALTER COLUMN area_m2 TYPE NUMERIC(18, 4) USING area_m2::NUMERIC(18, 4),
    ALTER COLUMN geometry TYPE public.geometry(Polygon, 4326)
        USING CASE
            WHEN geometry IS NULL THEN NULL
            ELSE ST_GeometryN(ST_CollectionExtract(geometry, 3), 1)::public.geometry(Polygon, 4326)
        END;

ALTER TABLE public.au3
    DROP COLUMN IF EXISTS name_desc,
    DROP COLUMN IF EXISTS old_code,
    DROP COLUMN IF EXISTS name_en;

CREATE INDEX IF NOT EXISTS idx_au3_geometry ON public.au3 USING gist (geometry);
CREATE INDEX IF NOT EXISTS idx_au3_au1_code ON public.au3 (au1_code);
CREATE INDEX IF NOT EXISTS idx_au3_au2_code ON public.au3 (au2_code);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au1'::regclass
           AND conname = 'au1_code_key'
    ) THEN
        ALTER TABLE public.au1
            ADD CONSTRAINT au1_code_key UNIQUE (code);
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au2'::regclass
           AND conname = 'au2_code_key'
    ) THEN
        ALTER TABLE public.au2
            ADD CONSTRAINT au2_code_key UNIQUE (code);
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au3'::regclass
           AND conname = 'au3_code_key'
    ) THEN
        ALTER TABLE public.au3
            ADD CONSTRAINT au3_code_key UNIQUE (code);
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au3'::regclass
           AND conname = 'au3_au1_code_fkey'
    ) THEN
        ALTER TABLE public.au3
            ADD CONSTRAINT au3_au1_code_fkey FOREIGN KEY (au1_code) REFERENCES public.au1(code);
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conrelid = 'public.au3'::regclass
           AND conname = 'au3_au2_code_fkey'
    ) THEN
        ALTER TABLE public.au3
            ADD CONSTRAINT au3_au2_code_fkey FOREIGN KEY (au2_code) REFERENCES public.au2(code);
    END IF;
END $$;
