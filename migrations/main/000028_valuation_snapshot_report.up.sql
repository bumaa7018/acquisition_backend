-- Хүчингүй болгосон үнэлгээний ТАЙЛАНГ snapshot дээр үлдээнэ.
--
-- ЯАГААД: үнэлгээ цуцлахад тухайн урсгалын assets / compensation мөрүүд
-- устдаг ба "Хөрөнгийн үнэлгээний тайлан" төрлийн parcel_document нь нэгж
-- талбарын ХАВСРАЛТААС хасагдана (дараагийн үнэлгээний тайлантай
-- андуурагдахгүйн тулд). Тайлангийн файл нь MinIO дээр ХЭВЭЭР үлдэх тул
-- түүхэн snapshot-оос татаж харах холбоос ЭНД хадгалагдана.
ALTER TABLE public.parcel_valuation_snapshot
    ADD COLUMN IF NOT EXISTS report_url text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS report_name text DEFAULT ''::text NOT NULL;

COMMENT ON COLUMN public.parcel_valuation_snapshot.report_url IS
    'Цуцлах үед хүчинтэй байсан үнэлгээний тайлангийн файлын зам (/api/files/...). Хоосон бол тайлангүй цуцлагдсан хуучин мөр.';
COMMENT ON COLUMN public.parcel_valuation_snapshot.report_name IS
    'Үнэлгээний тайлангийн харагдах нэр.';
