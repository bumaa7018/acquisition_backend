-- Нэгж талбарын төлөвийн ӨНГӨ.
--
-- ЯАГААД: төлөвийн өнгө одоогоор ГУРВАН газарт ХАТУУ бичигдсэн байна —
--   1) frontend `src/types/index.ts` (PARCEL_STATUS_STYLES, id-гаар)
--   2) frontend `src/components/map/layer-config.ts` (давхаргын өнгө)
--   3) GeoServer `styles/parcel_s0..s5.sld` (зурган дээрх бодит өнгө)
-- Иймд бүртгэлд ШИНЭ төлөв нэмэхэд өнгө нь хаанаас ч гарахгүй, саарал
-- өгөгдмөлөөр л харагддаг байв. Өнгийг бүртгэлийн өөрийнх нь шинж болгож,
-- гурвуулангийнх нь ЭХ СУРВАЛЖ болгоно.
--
-- NULL зөвшөөрөгдөнө: энэ migration нь зөвхөн schema (өгөгдөл өөрчлөхгүй).
-- Одоо байгаа мөрүүдийн өнгийг seeder тавина:
--     go run ./cmd/seed-sql -db main scripts/seed_parcel_status_color.sql
-- Өнгөгүй (NULL) төлөв нь frontend/SLD дээр саарал өгөгдмөлөөр зурагдана.
ALTER TABLE public.parcel_status
    ADD COLUMN IF NOT EXISTS color VARCHAR(7);

-- Зөвхөн `#rrggbb` хэлбэр. Энэ утга нь SLD-ийн `env` параметрээр GeoServer-ийн
-- зурагчид шууд очдог тул хэлбэрийг DB түвшинд бэхлэх нь чухал.
ALTER TABLE public.parcel_status
    DROP CONSTRAINT IF EXISTS parcel_status_color_hex;
ALTER TABLE public.parcel_status
    ADD CONSTRAINT parcel_status_color_hex
    CHECK (color IS NULL OR color ~ '^#[0-9a-fA-F]{6}$');
