-- Урьдчилан мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлсэн ТҮҮХ.
--
-- Төлөвлөгөөний 2.8.5 ажил: нөлөөлөлд өртсөн иргэнд E-Mongolia системээр
-- дамжуулан мэдэгдэл хүргүүлэх.
--
-- ЯАГААД parcel_notice_email-ээс ТУСДАА хүснэгт: хүлээн авагчийн ТАНИГЧ өөр
-- (имэйл хаяг биш, РЕГИСТРИЙН дугаар) ба гадаад системийн лавлах дугаар
-- (external_ref) нэмэлтээр хадгалагдана. Хоёуланг нэг хүснэгтэд шахвал
-- to_email нь регистрийн дугаар агуулах эсэх бүрхэг болно.
--
-- status:
--   'sent'   — гадаад систем хүлээн авсан
--   'mocked' — ГАДААД СИСТЕМ ХОЛБОГДООГҮЙ, мокоор бүртгэсэн (түр зуурын
--              хэрэгжүүлэлт: EMONGOLIA_BASE_URL тохируулаагүй үед). Ингэж
--              тэмдэглэснээр "иргэнд хүрсэн" гэж эндүүрэхгүй.
--   'failed' — гадаад систем татгалзсан (error_message-д шалтгаан)

CREATE TABLE IF NOT EXISTS public.parcel_emongolia_notice (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    -- register_no — хүлээн авагч иргэний регистрийн дугаар (E-Mongolia-ийн
    -- танигч). recipient_name нь зөвхөн хүн уншихад.
    register_no text NOT NULL,
    recipient_name text,
    subject text,
    file_url text,
    file_name text,
    size_bytes bigint,
    status text DEFAULT 'sent' NOT NULL,
    -- external_ref — гадаад системийн лавлагааны дугаар (мок үед MOCK-…).
    external_ref text,
    error_message text,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_by text
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_emongolia_notice_pkey') THEN
        ALTER TABLE ONLY public.parcel_emongolia_notice
            ADD CONSTRAINT parcel_emongolia_notice_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_emongolia_notice_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_emongolia_notice
            ADD CONSTRAINT parcel_emongolia_notice_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_emongolia_notice_status_check') THEN
        ALTER TABLE ONLY public.parcel_emongolia_notice
            ADD CONSTRAINT parcel_emongolia_notice_status_check
            CHECK (status IN ('sent', 'mocked', 'failed'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_parcel_emongolia_notice_parcel
    ON public.parcel_emongolia_notice USING btree (parcel_id, sent_at DESC);

COMMENT ON TABLE public.parcel_emongolia_notice IS
    'Мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлсэн түүх (регистрийн дугаараар, огноотой). status=mocked нь гадаад холболтгүй түр бүртгэл.';
