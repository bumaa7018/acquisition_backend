-- Чөлөөлөгдсөн газрын судалгаа (нэгж талбар тус бүрд).
--
-- Нөхөх олговор олгосны дараа тухайн нэгж талбарын үлдэгдэл иргэнээс
-- асууна: яндан, нүхэн жорлон, гэр сууцанд хамтран амьдрагч, мөн
-- нөлөөлөлд өртсөний дараа хаана оршин суусан.
--
-- parcel_id нь UNIQUE — нэгж талбарт НЭГ бүртгэл; дахин хадгалахад орлогдоно.
--
-- chimney_count / toilet_count / cohabitant_count: 1, 2, 3 (3 = «3+»).
-- Асуултад хариулаагүй бол NULL.
-- relocation_type: ger_district (Гэр хороолол) | apartment (Орон сууц) |
-- address (бусад хаяг — relocation_address-д бичнэ).

CREATE TABLE IF NOT EXISTS public.parcel_survey (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    chimney_count smallint,
    toilet_count smallint,
    cohabitant_count smallint,
    relocation_type text,
    relocation_address text,
    note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_survey_pkey') THEN
        ALTER TABLE ONLY public.parcel_survey
            ADD CONSTRAINT parcel_survey_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_survey_parcel_id_key') THEN
        ALTER TABLE ONLY public.parcel_survey
            ADD CONSTRAINT parcel_survey_parcel_id_key UNIQUE (parcel_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_survey_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_survey
            ADD CONSTRAINT parcel_survey_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_survey_counts_check') THEN
        ALTER TABLE ONLY public.parcel_survey
            ADD CONSTRAINT parcel_survey_counts_check CHECK (
                (chimney_count IS NULL OR chimney_count BETWEEN 1 AND 3)
                AND (toilet_count IS NULL OR toilet_count BETWEEN 1 AND 3)
                AND (cohabitant_count IS NULL OR cohabitant_count BETWEEN 1 AND 3)
            );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_survey_relocation_type_check') THEN
        ALTER TABLE ONLY public.parcel_survey
            ADD CONSTRAINT parcel_survey_relocation_type_check CHECK (
                relocation_type IS NULL OR relocation_type IN ('ger_district', 'apartment', 'address')
            );
    END IF;
END $$;

COMMENT ON TABLE public.parcel_survey IS
    'Чөлөөлөгдсөн газрын судалгаа: яндан, нойл, хамтран амьдрагчийн тоо, шилжин суурьшсан байршил. Нэгж талбарт нэг бүртгэл.';
