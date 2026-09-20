-- Хүчингүй болгосон үнэлгээний ЭХ ХҮСНЭГТ (импортын Excel) snapshot дээр үлдэнэ.
--
-- ЯАГААД: үнэлгээг оруулсан эх Excel нь "Үнэлгээний хүснэгт" (valuation_source)
-- төрлийн parcel_document болж хадгалагддаг. Үнэлгээ ХҮЧИНГҮЙ болоход тэр
-- баримт нэгж талбарын хавсралтад үлдвэл дараагийн үнэлгээний хүснэгттэй
-- андуурагдана — тайлантай ЯГ ижил асуудал (000028). Иймд баримтын мөрийг
-- хасаж, файлын холбоосыг ЭНД (түүхэн snapshot дээр) үлдээнэ: файл нь MinIO
-- дээр хэвээр тул "Дэлгэрэнгүй" цонхноос татагдана.
ALTER TABLE public.parcel_valuation_snapshot
    ADD COLUMN IF NOT EXISTS source_url text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS source_name text DEFAULT ''::text NOT NULL;

COMMENT ON COLUMN public.parcel_valuation_snapshot.source_url IS
    'Цуцлах үед хүчинтэй байсан үнэлгээний эх хүснэгтийн (Excel) зам (/api/files/...).';
COMMENT ON COLUMN public.parcel_valuation_snapshot.source_name IS
    'Эх хүснэгтийн харагдах нэр.';
