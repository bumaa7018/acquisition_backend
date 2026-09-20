-- Хүчингүй болсон үнэлгээний БҮХ файл ЗӨВХӨН тухайн snapshot-д харьяалагдана.
--
-- ЯАГААД: цуцлалтын мөчид үнэлгээнд хамаарах файлууд нь
--   1) "Баталгаажсан тайлан" (valuation_report төрлийн parcel_document),
--   2) "Тайлан" (илгээхэд хавсаргасан parcel_valuation_submission.draft_report_*),
--   3) "Үнэлгээний хүснэгт" (valuation_source төрлийн parcel_document),
--   4) хөрөнгийн зургууд (assets.photo_pdf_url — snapshot-ын assets jsonb дотор
--      хамт хуулагддаг тул энд давхардуулахгүй)
-- байдаг. Эдгээр нь нэгж талбарын хавсралт/илгээлтэд үлдвэл ДАРААГИЙН үнэлгээ
-- болон ӨӨР цуцлалтын дэлгэрэнгүй дээр гарч ирдэг байв.
--
-- ЯАГААД jsonb (тусдаа багана биш): файлын ТӨРӨЛ цаашид нэмэгддэг (тайлан,
-- хүснэгт, цуцлалтын үндэслэл...). Багана нэмэх бүрд migration + scan + DTO
-- гурвыг зэрэг өөрчлөхийн оронд snapshot бүр өөрийн бүрэн ЖАГСААЛТЫГ авч явна:
--   [{"label":"Баталгаажсан тайлан","name":"...","url":"/api/files/..."}]
-- Хуучин snapshot-ууд хоосон `[]`-тэй үлдэх тул frontend нь report_url/
-- source_url багануудаас уншдаг ХУУЧИН замаа хадгална (өгөгдөл засахгүй).
ALTER TABLE public.parcel_valuation_snapshot
    ADD COLUMN IF NOT EXISTS files jsonb DEFAULT '[]'::jsonb NOT NULL;

COMMENT ON COLUMN public.parcel_valuation_snapshot.files IS
    'Цуцлах мөчид энэ үнэлгээнд хамаарч байсан файлууд: [{label,name,url}]. Зөвхөн түүхийн дэлгэрэнгүйд харагдана.';
