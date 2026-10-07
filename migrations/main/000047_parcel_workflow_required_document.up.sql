-- Нэгж талбарын ажлын урсгалын ШИЛЖИЛТ БҮРД (parcel_workflow: хаанаас →
-- хаашаа) ЗААВАЛ байх хавсралтын төрөл. Нэг төлөвт өөр замаар ирэхэд өөр
-- баримт шаардаж болно (ж: «Үнэлгээ хийх → Чөлөөлсөн» — гэрээ; «Хүлээгдэж буй
-- → Чөлөөлсөн» — өмнөх шатуудын бүх баримт). Шилжихэд тухайн нэгж талбарт
-- (parcel_document) энэ төрлийн баримт байхгүй бол шилжихгүй.
-- Өгөгдмөл тохиргоог seeder (000047_parcel_workflow_required_documents) оруулна.
CREATE TABLE IF NOT EXISTS public.parcel_workflow_required_document (
    workflow_id      integer NOT NULL REFERENCES public.parcel_workflow(id) ON DELETE CASCADE,
    document_type_id integer NOT NULL REFERENCES public.document_type(id) ON DELETE CASCADE,
    created_at       timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (workflow_id, document_type_id)
);
