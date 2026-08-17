ALTER TABLE public.valuation_org_profile
    ALTER COLUMN id DROP DEFAULT;

DROP SEQUENCE IF EXISTS public.valuation_org_profile_id_seq;

ALTER TABLE public.valuation_org_profile
    DROP COLUMN IF EXISTS name;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'valuation_org_profile'
          AND column_name = 'id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'valuation_org_profile'
          AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE public.valuation_org_profile RENAME COLUMN id TO organization_id;
    END IF;
END $$;
