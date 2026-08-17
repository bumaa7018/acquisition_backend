-- Дроны ортофото нь файлын систем (MinIO) дээр хадгалагдана — API өөр дээрээ
-- файл хадгалахгүй. file_url нь тэр object-ийн зам.
--
-- file_name нь ХЭВЭЭР үлдэнэ: GeoServer-ийн ImageMosaic давхаргын `location`
-- атрибут (WMS-д CQL_FILTER-ээр шүүхэд) тэр нэрээр ажилладаг.
ALTER TABLE drone_image ADD COLUMN IF NOT EXISTS file_url TEXT NOT NULL DEFAULT '';
