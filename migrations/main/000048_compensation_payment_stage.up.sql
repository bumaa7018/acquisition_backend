-- «Нөхөх олговор олгосон баримт» — олговрын олголтын ШАТ ба ОГНОО.
--   document.payment_stage — тухайн баримт аль шатыг гэрчлэх: 60 | 40 | 100 (бүрэн)
--   document.payment_date  — олговор олгосон огноо (оруулахдаа өнөөдрөөр
--                            санал болгоно, хэрэглэгч засна)
--   parcel.compensation_paid_60/40/full    — нэгж талбарын олголтын гүйцэтгэл
--   parcel.compensation_paid_60/40/full_at — тухайн шатыг олгосон огноо
--     Тухайн шатны баримт нэгж талбарт байгаа эсэхээр тодорхойлогдоно (баримт
--     нэмэх/устгахад дахин бодогдоно). Эх хэвлэлийн «Төлбөрийн гүйцэтгэл»-д
--     урьд авсан санхүүжилт г.м. утгыг эндээс бодно.
-- Хавсралтын төрлийг seeder (000048_document_types_compensation_paid_receipt) нэмнэ.
ALTER TABLE public.document
    ADD COLUMN IF NOT EXISTS payment_stage smallint,
    ADD COLUMN IF NOT EXISTS payment_date date;
ALTER TABLE public.document DROP CONSTRAINT IF EXISTS document_payment_stage_check;
ALTER TABLE public.document ADD CONSTRAINT document_payment_stage_check
    CHECK (payment_stage IS NULL OR payment_stage IN (40, 60, 100));

ALTER TABLE public.parcel
    ADD COLUMN IF NOT EXISTS compensation_paid_60 boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS compensation_paid_40 boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS compensation_paid_full boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS compensation_paid_60_at date,
    ADD COLUMN IF NOT EXISTS compensation_paid_40_at date,
    ADD COLUMN IF NOT EXISTS compensation_paid_full_at date;
