-- Нэгж талбарын төлөв бүрийн ДҮРЭМ — өмнө кодонд төлөвийн нэр/дугаараар
-- хатуу бичигдсэн байсныг «Нэгж талбарын төлөв» тохиргооноос удирдана.
--   is_final           — эцсийн төлөв: нэгж талбарын засвар түгжигдэнэ;
--                        чөлөөлөлтийг баталгаажуулахад бүх нэгж талбар ийм байх
--   locks_progress     — энэ төлөвөөс цааш явц солихгүй
--   is_valuation_stage — үнэлгээний шат: гадаад байгууллага (мэрг./МИКА) харж,
--                        үнэлгээ засна; байгууллагад мэдэгдэл явна
--   is_released        — «Захирамжид холбогдоно»: захирамжийн төсөлд холбогдоно,
--                        шилжихэд батлагдсан олговор шаардана, чөлөөлөлт
--                        устгах / хил өөрчлөхийг хаана
--   requires_reason    — энэ төлөвт шилжихэд шалтгаан заавал
-- Өгөгдмөл утгыг seeder (000049_parcel_status_flags) өмнөх дүрмээр ононо.
ALTER TABLE public.parcel_status
    ADD COLUMN IF NOT EXISTS is_final boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS locks_progress boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS is_valuation_stage boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS is_released boolean DEFAULT false NOT NULL,
    ADD COLUMN IF NOT EXISTS requires_reason boolean DEFAULT false NOT NULL;
