-- ТӨСӨӨЛЛИЙН ҮНЭЛГЭЭНИЙ ӨӨРЧЛӨЛТИЙН ТҮҮХ (хавсралттай).
--
-- Асуудал: `parcel` дээр төсөөллийн үнэлгээний ЗӨВХӨН ХАМГИЙН СҮҮЛИЙН утга
-- (estimated_value / _confidence_percent / _base_fee_per_m2 / _at / _by)
-- хадгалагддаг. Иймд "хэн, хэзээ, ямар үнээр, ямар итгэлцүүрээр тооцсон"
-- гэдэг нь ДАРЖ БИЧИГДЭЖ, дараа нь хянах боломжгүй байв. Мөн үнэлгээг
-- гэрчлэх баримт (жишиг үнийн лавлагаа, зах зээлийн судалгаа) хавсаргах
-- газар байсангүй.
--
-- Тиймээс ӨӨРЧЛӨЛТ БҮР (тооцоолол ба арилгалт хоёул) энд НЭГ мөр болж
-- бичигдэнэ. `parcel`-ийн талбарууд нь "одоогийн байдал"-ын хурдан уншилт
-- хэвээр үлдэнэ — түүх нь тэднийг ЗАМЛАХГҮЙ, зөвхөн НЭМЖ бүртгэнэ.
--
-- Файл нь MinIO-д `estimated-value/<нэгж талбарын uuid>/<нэр>` түлхүүрээр
-- хадгалагдана; `attachment_url` нь `/api/files`-ийн ХАРЬЦАНГУЙ зам
-- (000016 миграцийн valuation-return-тай ижил зарчим).
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: хязгаарлалтууд БҮГД `CREATE TABLE IF NOT EXISTS`-ийн
-- ДОТОР зарлагдсан (`ALTER TABLE ... ADD CONSTRAINT` нь IF NOT EXISTS-ийг
-- дэмждэггүй тул хоёр дахь удаа ажиллахад миграци dirty болно — 000015-ийг үз).
CREATE TABLE IF NOT EXISTS public.parcel_estimated_value_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parcel_id uuid NOT NULL REFERENCES public.parcel(id) ON DELETE CASCADE,
    acquisition_id uuid NOT NULL REFERENCES public.land_acquisition(id) ON DELETE CASCADE,
    -- Тухайн үйлдлийн ДАРААХ үнэлгээ. NULL = үнэлгээг АРИЛГАСАН.
    value numeric(18,2),
    -- Дуудлага худалдааны анхны үнийн итгэлцүүр. NULL = итгэлцүүр
    -- ХЭРЭГЛЭЭГҮЙ (өмчлөх эрхийн газар — суурь үнэ нь зах зээлийн жишиг үнэ).
    confidence_percent numeric(18,8),
    -- Гараар оруулсан суурь үнэ (₮/м²). ГУС-ийн бүсийн төлбөрөөр тооцсон бол NULL.
    base_fee_per_m2 numeric(18,4),
    -- Тооцоололд хэрэглэсэн НӨЛӨӨЛӨЛД ӨРТСӨН талбай. Дараа нь талбай
    -- өөрчлөгдвөл хуучин дүн яагаад тэр байсныг ЭНЭ баганаас уншина.
    area_m2 numeric(18,4) DEFAULT 0 NOT NULL,
    attachment_url text DEFAULT ''::text NOT NULL,
    attachment_name text DEFAULT ''::text NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parcel_estimated_value_history_pkey PRIMARY KEY (id)
);

-- Нэгж талбарын түүхийг шинэ мөрөөс хуучин руу уншина.
CREATE INDEX IF NOT EXISTS idx_parcel_estimated_value_history_parcel
    ON public.parcel_estimated_value_history (parcel_id, created_at DESC);

COMMENT ON TABLE public.parcel_estimated_value_history IS
    'Төсөөллийн үнэлгээний өөрчлөлт бүрийн түүх (хавсралттай). parcel дээрх утга нь зөвхөн одоогийн байдал.';
COMMENT ON COLUMN public.parcel_estimated_value_history.value IS
    'Үйлдлийн дараах үнэлгээ (₮). NULL = арилгасан.';
COMMENT ON COLUMN public.parcel_estimated_value_history.confidence_percent IS
    'Итгэлцүүр (%). NULL = хэрэглээгүй (өмчлөх эрхийн газар).';
COMMENT ON COLUMN public.parcel_estimated_value_history.attachment_url IS
    'Хавсралт (/api/files харьцангуй зам). Хоосон = хавсралтгүй.';
