-- Газрын ТӨЛБӨРИЙН БОДОЛТ ба суурь үнийн ХҮЧИН ЗҮЙЛ — ГУС-аас татагдсан
-- мэдээллийг хадгалах хүснэгтүүд.
--
-- parcel_fee: POST /parcel/fee-гийн items (төлбөрийн бүс тус бүрийн бодолт).
--   Нийт төлбөр нь payment-ийн НИЙЛБЭР тул header хүснэгт үүсгэсэнгүй —
--   давхардсан нийлбэрийг хоёр газар зэрэг зөв байлгах шаардлага гарахгүй.
--   Бүсийн мөрөнд эх системд тогтвортой түлхүүр байхгүй (zone_id хоосон байж
--   болно) тул item_no-гоор эрэмбэлж, sync үед БҮХЭЛДЭЭ орлуулна.
--
-- parcel_base_price_factor: GET /parcel/base-price/{parcel}/factors-ийн
--   factor_values. Суурь үнэ өөрөө parcel_detail.base_price_per_ha-д бичигдэнэ.

CREATE TABLE IF NOT EXISTS public.parcel_fee (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    -- item_no — ГУС-аас ирсэн дараалал (1-ээс). Эрэмбэ, давхардлын түлхүүр.
    item_no integer NOT NULL,
    zone_id text,
    zone_type text,
    zone_no text,
    zone_name text,
    area numeric(20,2),
    landuse_area numeric(20,2),
    zone_area numeric(20,2),
    base_fee_id text,
    resolution_id text,
    confidence_percent numeric(12,4),
    subsidized_area numeric(20,2),
    subsidized_fee_rate numeric(12,4),
    base_price_m2 numeric(20,2),
    base_fee_per_m2 numeric(20,4),
    landuses text,
    value numeric(20,4),
    payment numeric(20,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.parcel_base_price_factor (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    factor_id integer NOT NULL,
    factor_code text,
    factor_name text,
    factor_data_type text,
    factor_value numeric(20,4),
    year_value integer,
    in_active boolean,
    -- closets_parcel_id — эх хүснэгтийн баганын нэр ЯГ ингэж бичигдсэн
    -- (closets ≠ closest). ГУС-тай зөрөхгүйн тулд ЗАСААГҮЙ.
    closets_parcel_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_fee_pkey') THEN
        ALTER TABLE ONLY public.parcel_fee
            ADD CONSTRAINT parcel_fee_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_fee_parcel_id_item_no_key') THEN
        ALTER TABLE ONLY public.parcel_fee
            ADD CONSTRAINT parcel_fee_parcel_id_item_no_key UNIQUE (parcel_id, item_no);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_fee_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_fee
            ADD CONSTRAINT parcel_fee_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_base_price_factor_pkey') THEN
        ALTER TABLE ONLY public.parcel_base_price_factor
            ADD CONSTRAINT parcel_base_price_factor_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_base_price_factor_parcel_id_factor_id_key') THEN
        ALTER TABLE ONLY public.parcel_base_price_factor
            ADD CONSTRAINT parcel_base_price_factor_parcel_id_factor_id_key UNIQUE (parcel_id, factor_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parcel_base_price_factor_parcel_id_fkey') THEN
        ALTER TABLE ONLY public.parcel_base_price_factor
            ADD CONSTRAINT parcel_base_price_factor_parcel_id_fkey
            FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_parcel_fee_parcel
    ON public.parcel_fee USING btree (parcel_id);
CREATE INDEX IF NOT EXISTS idx_parcel_base_price_factor_parcel
    ON public.parcel_base_price_factor USING btree (parcel_id);

COMMENT ON TABLE public.parcel_fee IS
    'Газрын төлбөрийн бодолт бүсээр (ГУС POST /parcel/fee). Нийт төлбөр = payment-ийн нийлбэр.';
COMMENT ON TABLE public.parcel_base_price_factor IS
    'Газрын суурь үнийн үнэлгээний хүчин зүйл (ГУС GET /parcel/base-price/{parcel}/factors).';
