-- Нийгэм, эдийн засгийн судалгааны ТООН мэдээлэл (чөлөөлөлт тус бүрд).
--
-- Төлөвлөгөөний 2.4.2.1 ажил: нөлөөлөлд өртсөн иргэн, хуулийн этгээдээс авсан
-- судалгааны дүнг цахим системд бүртгэж, судалгааны эх баримтыг (PDF)
-- хавсаргана.
--
-- acquisition_id нь UNIQUE — чөлөөлөлтөд НЭГ бүртгэл байна. Дахин "Мэдээлэл
-- оруулах" үед мөр ОРЛОГДОНО (дарагдана) ба солигдсон хуучин файл нь
-- хадгалалтаас цэвэрлэгдэнэ (орфон файл үлдэхгүй).
--
-- Судалгааны файл нь ЗААВАЛ байх тул file_url NOT NULL. Хэрэглэгч олон PDF
-- сонгосон бол browser дээр НЭГ PDF болгож нэгтгэгдэнэ (pdf-lib) — сервер рүү
-- үргэлж нэг файл ирнэ.

CREATE TABLE IF NOT EXISTS public.acquisition_socio_survey (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    acquisition_id uuid NOT NULL,
    -- agreed_count / rejected_count — зөвшөөрсөн / татгалзсан иргэн, хуулийн
    -- этгээдийн ТОО (судалгааны дүн).
    agreed_count integer DEFAULT 0 NOT NULL,
    rejected_count integer DEFAULT 0 NOT NULL,
    note text,
    -- file_url — MinIO дээрх судалгааны PDF-ийн харьцангуй зам (/api/files/...)
    file_url text NOT NULL,
    file_name text,
    size_bytes bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'acquisition_socio_survey_pkey') THEN
        ALTER TABLE ONLY public.acquisition_socio_survey
            ADD CONSTRAINT acquisition_socio_survey_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'acquisition_socio_survey_acquisition_id_key') THEN
        ALTER TABLE ONLY public.acquisition_socio_survey
            ADD CONSTRAINT acquisition_socio_survey_acquisition_id_key UNIQUE (acquisition_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'acquisition_socio_survey_acquisition_id_fkey') THEN
        ALTER TABLE ONLY public.acquisition_socio_survey
            ADD CONSTRAINT acquisition_socio_survey_acquisition_id_fkey
            FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'acquisition_socio_survey_counts_check') THEN
        ALTER TABLE ONLY public.acquisition_socio_survey
            ADD CONSTRAINT acquisition_socio_survey_counts_check
            CHECK (agreed_count >= 0 AND rejected_count >= 0);
    END IF;
END $$;

COMMENT ON TABLE public.acquisition_socio_survey IS
    'Нийгэм, эдийн засгийн судалгааны дүн (зөвшөөрсөн/татгалзсан тоо, тайлбар) + судалгааны PDF. Чөлөөлөлтөд нэг бүртгэл.';
