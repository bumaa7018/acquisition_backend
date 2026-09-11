-- ЧӨЛӨӨЛӨЛТИЙН НИТХ ТОГТООЛ БА БҮЛЭГ ЖАГСААЛТЫН ДУГААР.
--
-- Газар чөлөөлөх ажиллагаа НИТХ-ын (Нийслэлийн иргэдийн Төлөөлөгчдийн Хурал)
-- тогтоолоор баталгаажиж, тогтоолын хавсралтад чөлөөлөгдөх талбарууд БҮЛЭГ
-- ЖАГСААЛТААР дугаарлагдан орсон байдаг. Эдгээр хоёр дугаарыг чөлөөлөлт тус
-- бүрд гараас бүртгэж, дэлгэрэнгүй мэдээлэлд харуулна.
--
-- Одоо байгаа `decree_number`/`decree_date` нь НЗД-ын ЗАХИРАМЖ (өөр баримт) —
-- түүнийг орлуулахгүй, зэрэгцэн хадгалагдана.
--
-- Хоосон мөр = бүртгээгүй ('' DEFAULT, decree_number-тай ижил хэв маяг).
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: `ADD COLUMN IF NOT EXISTS` нь идемпотент.
ALTER TABLE public.land_acquisition
    ADD COLUMN IF NOT EXISTS nith_decree_number character varying(100) DEFAULT ''::character varying NOT NULL,
    ADD COLUMN IF NOT EXISTS group_list_number character varying(100) DEFAULT ''::character varying NOT NULL;

COMMENT ON COLUMN public.land_acquisition.nith_decree_number IS
    'НИТХ (Нийслэлийн иргэдийн Төлөөлөгчдийн Хурал) тогтоолын дугаар. '''' = бүртгээгүй.';
COMMENT ON COLUMN public.land_acquisition.group_list_number IS
    'НИТХ тогтоолын хавсралтын бүлэг жагсаалтын дугаар. '''' = бүртгээгүй.';
