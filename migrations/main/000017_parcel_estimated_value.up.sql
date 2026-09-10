-- НЭГЖ ТАЛБАРЫН ТӨСӨӨЛЛИЙН ҮНЭЛГЭЭ.
--
-- Мэргэжилтэн/админ ГАРААС оруулдаг урьдчилсан тооцоо: албан ёсны үнэлгээ
-- (land_valuation, assets) хийгдэхээс өмнө төсөв төлөвлөх, хэлэлцээрийн
-- жишиг тогтооход хэрэглэгдэнэ. Мэргэжлийн байгууллагад ЗӨВХӨН ХАРАГДАНА.
--
-- ЯАГААД `parcel` дээр (parcel_detail дээр БИШ):
-- `parcel_detail` нь SyncParcel үед БҮХЭЛДЭЭ орлуулагддаг (хуучин мөр устаж
-- шинээр үүсдэг) тул гараас оруулсан утга ГУС-аас дахин татах бүрд УСТАНА.
-- `parcel` дээрх хэрэглэгчийн талбарууд (db_changed, changed_parcel_id,
-- compensation_paid) sync-д хөндөгддөггүй — төсөөллийн үнэлгээ мөн тэнд байна.
--
-- NULL = оруулаагүй (0 гэсэн утгатай ялгаатай: 0 нь "үнэлгээгүй" гэсэн
-- санаатай оруулсан утга байж болно).
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: `ADD COLUMN IF NOT EXISTS` нь идемпотент.
ALTER TABLE public.parcel
    ADD COLUMN IF NOT EXISTS estimated_value numeric(20,2),
    ADD COLUMN IF NOT EXISTS estimated_value_at timestamp with time zone,
    ADD COLUMN IF NOT EXISTS estimated_value_by text DEFAULT ''::text NOT NULL;

COMMENT ON COLUMN public.parcel.estimated_value IS
    'Төсөөллийн үнэлгээ (₮), мэргэжилтэн/админ гараас оруулна. NULL = оруулаагүй.';
COMMENT ON COLUMN public.parcel.estimated_value_at IS
    'Төсөөллийн үнэлгээг сүүлд оруулсан/зассан хугацаа.';
COMMENT ON COLUMN public.parcel.estimated_value_by IS
    'Төсөөллийн үнэлгээг сүүлд оруулсан/зассан хэрэглэгчийн ID.';
