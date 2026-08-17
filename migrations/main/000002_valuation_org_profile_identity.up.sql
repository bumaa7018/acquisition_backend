-- Мэргэжлийн байгууллагын identity нь appdb-ийн өөрийн profile хүснэгтэд байна.
-- sdplatform.sd_organization нь ердийн HR лавлахын эх сурвалж хэвээр үлдэнэ.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'valuation_org_profile'
          AND column_name = 'organization_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'valuation_org_profile'
          AND column_name = 'id'
    ) THEN
        ALTER TABLE public.valuation_org_profile RENAME COLUMN organization_id TO id;
    END IF;
END $$;

ALTER TABLE public.valuation_org_profile
    ADD COLUMN IF NOT EXISTS name varchar(250) NOT NULL DEFAULT '';

CREATE SEQUENCE IF NOT EXISTS public.valuation_org_profile_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.valuation_org_profile_id_seq
    OWNED BY public.valuation_org_profile.id;

ALTER TABLE public.valuation_org_profile
    ALTER COLUMN id SET DEFAULT nextval('public.valuation_org_profile_id_seq'::regclass);

SELECT setval(
    'public.valuation_org_profile_id_seq'::regclass,
    COALESCE(MAX(id), 0) + 1,
    false
)
FROM public.valuation_org_profile;

-- Хуучин profile-ууд нэрээ тусад нь хадгалаагүй байсан тул боломжтой
-- fallback-ийг app-owned short_name-оос авна. sdplatform руу хандахгүй.
UPDATE public.valuation_org_profile
SET name = short_name
WHERE name = '' AND short_name <> '';
