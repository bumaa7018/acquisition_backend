-- Үнэлгээний хүснэгт бүрийн ТАЙЛБАР.
--
-- ЯАГААД: үнэлгээний шинэ загвар (нэг хуудсан "Үнэлгээний хүснэгт") дээр хүснэгт
-- бүрийн доор "ТАЙЛБАР:" мөр байдаг — газрын суурь үнийг юугаар тодотгосон, элэгдлийн
-- хувийг хэрхэн тогтоосон, ямар зардал юунд хамаарах гэх мэт ҮНДЭСЛЭЛ тэнд бичигддэг.
-- Энэ бол үнэлгээг хянах, маргаан шийдвэрлэхэд хамгийн чухал мэдээлэл боловч урьд нь
-- хадгалагддаггүй, Excel-ээс уншаад хаядаг байв.
--
-- ХААНА: land_valuation дээр багана болгож нэмэх боломжтой ч, тэр мөрийг устгахад
-- (Газрын үнэлгээ → Устгах) БУСАД хүснэгтийн тайлбар хамт алга болно. Тиймээс
-- тайлбарыг бие даасан хүснэгтэд, (чөлөөлөлт, нэгж талбар, урсгал, хэсэг) түлхүүрээр
-- хадгална. Урсгал (valuation_type) тус бүр өөрийн тайлбартай.
--
-- section_key нь frontend-ийн ValuationSectionKey-тэй ЯГ ижил утгатай:
--   property_desc, land_legal, land_valuation, building_spec, building_cost,
--   other_assets, temporary_cost, clearance_cost, lost_income, summary,
--   conclusion, certification
-- Бүртгэл болгож хатуулаагүй — загварт шинэ хүснэгт нэмэгдэхэд миграц шаардахгүй.
CREATE TABLE IF NOT EXISTS public.valuation_section_note (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL,
    section_key text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT valuation_section_note_pkey PRIMARY KEY (id)
);

-- Нэг талбар + урсгал + хэсэгт ЯГ нэг тайлбар (upsert-ийн түлхүүр).
CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_section_note
    ON public.valuation_section_note
    USING btree (acquisition_id, parcel_id, valuation_type, section_key);

-- Нэгж талбарын дэлгэц нэг дуудлагаар бүх тайлбараа татдаг тул жагсаалтын индекс.
CREATE INDEX IF NOT EXISTS idx_valuation_section_note_parcel
    ON public.valuation_section_note
    USING btree (acquisition_id, parcel_id, valuation_type);

COMMENT ON TABLE public.valuation_section_note IS
    'Үнэлгээний хүснэгт (3.1–5.1) бүрийн ТАЙЛБАР. Excel-ээс импортлогдох ба гараас засагдана.';
COMMENT ON COLUMN public.valuation_section_note.section_key IS
    'Хүснэгтийн түлхүүр (frontend ValuationSectionKey-тэй ижил): land_valuation, building_cost, …';
