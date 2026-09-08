-- УБЕГ-ийн (Улсын бүртгэлийн ерөнхий газар) газар өмчлөлийн бүртгэл.
--
-- Эх сурвалж: XYP гарц GET /api/geo/xyp/ownership/citizen/parcel?parcel_id=...
--
-- Нэг нэгж талбар дээр өмчлөгч ОЛОН (хамтран өмчлөгчид) бөгөөд өмчлөлийн
-- ТҮҮХ (өв залгамжлал, өмчлөгч өөрчлөгдөх) мөр мөрөөр ирдэг: тиймээс нэг
-- бүртгэлийн дугаар (registr) хэд хэдэн огноотой давтагдаж болно.
--
-- Эх системд мөрийн ТОГТВОРТОЙ түлхүүр БАЙХГҮЙ (өмчлөлийн бүртгэлийн id
-- буцдаггүй) тул parcel_fee-тэй ижил зарчмаар item_no-гоор эрэмбэлж, sync
-- үед нэгж талбарын мөрүүдийг БҮХЭЛДЭЭ орлуулна.

CREATE TABLE IF NOT EXISTS public.parcel_ownership (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    -- item_no — УБЕГ-аас ирсэн дараалал (1-ээс). Эрэмбэ, давхардлын түлхүүр.
    item_no integer NOT NULL,
    -- source_parcel_id — prprty_parcel_id (УБЕГ дээрх нэгж талбарын дугаар).
    -- Манай parcel.parcel_id-тай ижил байх ёстой ч зөрүү гарвал харагдана.
    source_parcel_id text,
    -- record_date — ognoo. YYYYMMDD хэлбэрээр ЯГ ирсэн байдлаараа хадгална
    -- (цагийн бүсийн хөрвүүлэлтээр огноо шилжихээс сэргийлж text).
    record_date text,
    -- certificate_no — ubd (гэрчилгээ/улсын бүртгэлийн дугаар).
    certificate_no text,
    -- landuse_name — zoriulalt (ж: "Гэр бүлийн хэрэгцээний").
    landuse_name text,
    -- area — hemjee (эх системд текстээр ирдэг; тоо болгож чадвал бичнэ).
    area numeric(20,2),
    register_no text,
    first_name text,
    last_name text,
    -- ownership_type / ownership_code — umchlugch / umch_code
    -- (ж: "Өв залгамжлуулсан" / 000005, "Өмчилсөн" / 000001).
    ownership_type text,
    ownership_code text,
    -- service_name / service_code — uilchilgee / service_code, өмчлөлийн
    -- бүртгэлийг үүсгэсэн улсын бүртгэлийн үйлчилгээ.
    service_name text,
    service_code text,
    aimag_city_name text,
    aimag_code text,
    soum_district_name text,
    soum_code text,
    bag_khoroo_name text,
    bag_code text,
    address_street_name text,
    bair_number text,
    toot text,
    full_address text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_ownership_pkey') THEN
        ALTER TABLE ONLY public.parcel_ownership
            ADD CONSTRAINT parcel_ownership_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_ownership_parcel_id_item_no_key') THEN
        ALTER TABLE ONLY public.parcel_ownership
            ADD CONSTRAINT parcel_ownership_parcel_id_item_no_key UNIQUE (parcel_id, item_no);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_ownership_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_ownership
            ADD CONSTRAINT parcel_ownership_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_parcel_ownership_parcel
    ON public.parcel_ownership USING btree (parcel_id);

COMMENT ON TABLE public.parcel_ownership IS
    'УБЕГ-ийн газар өмчлөлийн бүртгэл (XYP GET /xyp/ownership/citizen/parcel). Sync үед нэгж талбарын мөрүүд бүхэлдээ орлогдоно.';
