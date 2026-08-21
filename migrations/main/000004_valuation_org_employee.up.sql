-- Үнэлгээний байгууллагын АЖИЛТАН — appdb-ийн өөрийн хүснэгт.
--
-- ЯАГААД sdplatform-ийн HR бүтцийг хэрэглэхгүй вэ:
--
-- Ажилтан sdplatform дээр байхын тулд байгууллага нь sd_organization дээр байх
-- шаардлагатай (hr_employee → hr_department → hr_department.organization).
-- Гэвч 000002 миграцаар байгууллагын identity нь appdb-ийн
-- valuation_org_profile.id (өөрийн sequence) болсон тул хоёр ID орон зай
-- салсан: sd_organization.id ≠ valuation_org_profile.id.
--
-- Үүний улмаас JWT-ийн org_id (hr_department.organization-оос уншигддаг) нь
-- land_acquisition.professional_org_id / parcel.independent_org_id (эдгээр нь
-- valuation_org_profile.id-г заадаг)-тай ХЭЗЭЭ Ч таарахгүй байсан — өөрөөр
-- хэлбэл үнэлгээний байгууллагын ажилтан нэвтэрсэн ч өөрт оноогдсон
-- чөлөөлөлтөө харж чаддаггүй байв.
--
-- Ажилтныг appdb-д авчирснаар org_id = valuation_org_profile.id болж
-- professional_org_id / independent_org_id-тай ШУУД тулгагдана.
--
-- Нэвтрэх эрх нь sdplatform.sd_user хэвээр (authentication тэндээс уншина) —
-- өөр САН тул user_id дээр гадаад түлхүүр тавих боломжгүй. sdplatform-ийн
-- бүтэц ХӨНДӨГДӨХГҮЙ: тэнд зөвхөн sd_user / sd_user_role МӨР нэмэгдэнэ.

CREATE SEQUENCE IF NOT EXISTS public.valuation_org_employee_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.valuation_org_employee (
    id integer DEFAULT nextval('public.valuation_org_employee_id_seq'::regclass) NOT NULL,
    org_id integer NOT NULL,
    -- sdplatform.sd_user.user_id. NULL бол нэвтрэх эрхгүй ажилтан.
    user_id integer,
    last_name character varying(250) DEFAULT ''::character varying NOT NULL,
    first_name character varying(250) DEFAULT ''::character varying NOT NULL,
    register_no character varying(20) DEFAULT ''::character varying NOT NULL,
    position_name character varying(250) DEFAULT ''::character varying NOT NULL,
    phone character varying(50) DEFAULT ''::character varying NOT NULL,
    email character varying(250) DEFAULT ''::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE public.valuation_org_employee_id_seq
    OWNED BY public.valuation_org_employee.id;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'valuation_org_employee_pkey'
    ) THEN
        ALTER TABLE ONLY public.valuation_org_employee
            ADD CONSTRAINT valuation_org_employee_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'valuation_org_employee_org_fkey'
    ) THEN
        ALTER TABLE ONLY public.valuation_org_employee
            ADD CONSTRAINT valuation_org_employee_org_fkey
            FOREIGN KEY (org_id) REFERENCES public.valuation_org_profile(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_valuation_org_employee_org
    ON public.valuation_org_employee USING btree (org_id);

-- Нэг нэвтрэх эрх ЗӨВХӨН нэг ИДЭВХТЭЙ ажилтанд харьяалагдана — эс бөгөөс
-- токенд ямар org_id тавихыг шийдэх боломжгүй болж, өөр байгууллагын өгөгдөл
-- рүү хандах эрсдэл үүснэ (OrgIDForUser нь is_active-аар шүүдэг).
--
-- is_active нөхцөл нь ЗАЙЛШГҮЙ: ажилтныг хасахад мөр УСТАХГҮЙ, зөвхөн идэвхгүй
-- болж user_id-г эзэлсээр байдаг. Нөхцөлгүй бол тэр хүн өөр байгууллагад (мөн
-- хуучин байгууллагадаа) дахин бүртгэгдэж чадахгүй байв.
CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_org_employee_user
    ON public.valuation_org_employee USING btree (user_id)
    WHERE (user_id IS NOT NULL AND is_active);

-- Нэг байгууллагын дотор нэг хүн (регистрээр) хоёр удаа бүртгэгдэхгүй.
CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_org_employee_register
    ON public.valuation_org_employee USING btree (org_id, register_no)
    WHERE ((register_no)::text <> ''::text);

COMMENT ON TABLE public.valuation_org_employee IS
    'Үнэлгээний байгууллагын ажилтан. user_id нь sdplatform.sd_user.user_id (өөр сан тул FK байхгүй).';
COMMENT ON COLUMN public.valuation_org_employee.user_id IS
    'Нэвтрэх эрх (sdplatform.sd_user.user_id). NULL бол ажилтан системд нэвтрэхгүй.';
COMMENT ON COLUMN public.valuation_org_employee.is_active IS
    'Идэвхгүй ажилтан нэвтрэхгүй ба байгууллагын ажлыг харахгүй.';
