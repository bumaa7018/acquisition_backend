-- Нэгж талбарын төлөвийн ӨНГӨ — одоо байгаа 6 төлөвт утга олгоно.
--
-- Migration (000032) нь зөвхөн баганыг нэмнэ (NULL). Энэ нь ӨГӨГДӨЛ тул
-- seeder алхмаар ажиллана:
--     go run ./cmd/seed-sql -db main scripts/seed_parcel_status_color.sql
--
-- ТЭМДЭГЛЭЛ: асаах үеийн seeder (internal/seed/seed_reference.go) нь одоо
-- төлөвийг ӨНГӨТЭЙ нь үүсгэж, өнгөгүй мөрийг бөглөдөг болсон тул `make seed` /
-- `make reset-db` хийхэд энэ скрипт ШААРДЛАГАГҮЙ. Зөвхөн өнгийг гараар
-- анхны утгад нь буцаах хэрэгтэй үед гар аргаар ажиллуулна.
--
-- Утгууд нь ӨМНӨ НЬ frontend-д хатуу бичигдсэн байсантай ЯГ ижил
-- (src/types/index.ts → PARCEL_STATUS_STYLES, layer-config.ts, болон
-- GeoServer styles/parcel_s0..s5.sld). Иймд харагдах байдал өөрчлөгдөхгүй —
-- зөвхөн эх сурвалж нь бүртгэл рүү шилжинэ.
--
-- Давтан ажиллуулж болно. Гараар өөрчилсөн өнгийг ДАРЖ БИЧИХГҮЙ —
-- зөвхөн NULL байгааг нь бөглөнө.
UPDATE public.parcel_status SET color = v.color
FROM (VALUES
    ('pending',      '#64748b'),  -- Хүлээгдэж буй — саарал
    ('processing',   '#facc15'),  -- Зөвшилцөх шатандаа — тод шар
    ('evaluating',   '#f97316'),  -- Үнэлгээ хийх — улбар шар
    ('affected_out', '#3b82f6'),  -- Нөлөөлөгдсөн гарсан — цэнхэр
    ('rejected',     '#ef4444'),  -- Татгалзсан — улаан
    ('released',     '#22c55e')   -- Чөлөөлсөн — ногоон
) AS v(code, color)
WHERE public.parcel_status.code = v.code
  AND public.parcel_status.color IS NULL;

SELECT id, code, name, color FROM public.parcel_status ORDER BY sort_order, id;
