-- Урьдчилан мэдэгдэх хуудсыг ИМЭЙЛЭЭР илгээсэн ТҮҮХ.
--
-- Төлөвлөгөөний 2.4.2.2 ажил: мэдэгдэх хуудсыг цахим хэлбэрээр хүргүүлэх.
-- Нэгж талбар дээр илгээлт ОЛОН удаа гарч болно (өөр эзэмшигч, солигдсон
-- имэйл, дахин илгээх) тул мөр бүр НЭГ илгээлт — түүх хэлбэрээр хуримтлагдана.
--
-- status:
--   'sent'   — SMTP сервер хүлээн авсан
--   'failed' — илгээхэд алдаа (error_message-д шалтгаан). Алдаатай оролдлогыг ч
--              бүртгэнэ: "илгээсэн гэж бодсон" гэх эндүүрлээс сэргийлнэ.
--
-- file_url — илгээсэн PDF нь MinIO-д `notice/<parcel_id>/…` гэж хадгалагдана.
-- Ингэснээр яг ямар агуулга илгээснийг дараа нь дахин нээж харж болно.

CREATE TABLE IF NOT EXISTS public.parcel_notice_email (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    to_email text NOT NULL,
    subject text,
    file_url text,
    file_name text,
    size_bytes bigint,
    status text DEFAULT 'sent' NOT NULL,
    error_message text,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_by text
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_notice_email_pkey') THEN
        ALTER TABLE ONLY public.parcel_notice_email
            ADD CONSTRAINT parcel_notice_email_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_notice_email_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_notice_email
            ADD CONSTRAINT parcel_notice_email_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_notice_email_status_check') THEN
        ALTER TABLE ONLY public.parcel_notice_email
            ADD CONSTRAINT parcel_notice_email_status_check
            CHECK (status IN ('sent', 'failed'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_parcel_notice_email_parcel
    ON public.parcel_notice_email USING btree (parcel_id, sent_at DESC);

COMMENT ON TABLE public.parcel_notice_email IS
    'Урьдчилан мэдэгдэх хуудсыг имэйлээр илгээсэн түүх (нэгж талбар тус бүрд, огноотой).';
