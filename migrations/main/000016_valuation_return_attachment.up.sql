-- ҮНЭЛГЭЭНИЙ ТАЙЛАНГ БУЦААХАД ХАВСРАЛТ.
--
-- Санхүүгийн мэргэжилтэн үнэлгээг буцаахдаа ЗААВАЛ тайлбар бичдэг. Гэвч
-- тайлбар нь ихэвчлэн хүрэлцдэггүй: залруулга шаардсан хуудсыг тэмдэглэсэн
-- сканнер, дүнгийн зөрүүг харуулсан хүснэгт зэргийг ХАВСРАХ шаардлагатай.
-- Хавсралт нь ЗААВАЛ БИШ — тайлбар нь өмнөх шигээ ганцаараа хүрнэ.
--
-- ХОЁР хүснэгтэд нэмэгдэнэ:
--   parcel_valuation_submission          — ХАМГИЙН СҮҮЛИЙН үйлдлийн хавсралт.
--     `last_note` яг ижил зарчмаар ажилладаг: төлөвийн картан дээр тайлбартай
--     хамт хавсралтын холбоос гарна (түүхийг ачаалахгүйгээр).
--   parcel_valuation_submission_history  — үйлдэл ТУС БҮРийн хавсралт. Түүх нь
--     эх сурвалж: дараагийн буцаалт `last_note`-ыг дарж бичсэн ч хуучин
--     хавсралт түүхэн дээрээ үлдэнэ.
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: `ADD COLUMN IF NOT EXISTS` нь идемпотент (ADD CONSTRAINT-аас
-- ялгаатай — 000015-ийн тайлбарыг үз).
--
-- Файл нь MinIO-д `valuation-return/<нэгж талбарын uuid>/<нэр>` түлхүүрээр
-- хадгалагдана; `attachment_url` нь `/api/files`-ийн ХАРЬЦАНГУЙ зам.
ALTER TABLE public.parcel_valuation_submission
    ADD COLUMN IF NOT EXISTS attachment_url text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS attachment_name text DEFAULT ''::text NOT NULL;

ALTER TABLE public.parcel_valuation_submission_history
    ADD COLUMN IF NOT EXISTS attachment_url text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS attachment_name text DEFAULT ''::text NOT NULL;

COMMENT ON COLUMN public.parcel_valuation_submission.attachment_url IS
    'Сүүлийн үйлдлийн хавсралт (/api/files харьцангуй зам). Хоосон = хавсралтгүй.';
COMMENT ON COLUMN public.parcel_valuation_submission_history.attachment_url IS
    'Тухайн үйлдлийн хавсралт (/api/files харьцангуй зам). Хоосон = хавсралтгүй.';
