-- Нэгж талбар үүсэхэд чөлөөлөгдсөн газрын судалгаа (хоосон) АВТОМАТААР үүснэ.
--
-- Нэгж талбар хэд хэдэн газраас нэмэгддэг (ГУС-аас татах insertParcelRow,
-- хуучин мэдээллийн импорт) тул кодонд биш, TRIGGER-ээр баталгаажуулна —
-- шинээр нэмэгдэх замууд ч мартагдахгүй.
--
-- FOR EACH STATEMENT + transition table: бөөнөөр (импорт) нэмэхэд мөр бүрд
-- тусдаа INSERT хийхгүй, нэг INSERT ... SELECT болно. ON CONFLICT DO NOTHING
-- нь parcel INSERT ... ON CONFLICT DO NOTHING-ийн алгассан мөрийг (new_rows-д
-- орохгүй) болон аль хэдийн байгаа судалгааг хамгаална.

CREATE OR REPLACE FUNCTION public.parcel_survey_auto_create()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO public.parcel_survey (parcel_id)
    SELECT n.id FROM new_rows n
    ON CONFLICT (parcel_id) DO NOTHING;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_parcel_survey_auto_create ON public.parcel;
CREATE TRIGGER trg_parcel_survey_auto_create
    AFTER INSERT ON public.parcel
    REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION public.parcel_survey_auto_create();

-- Өмнө үүссэн нэгж талбаруудад хоосон судалгааг нөхнө.
INSERT INTO public.parcel_survey (parcel_id)
SELECT p.id FROM public.parcel p
ON CONFLICT (parcel_id) DO NOTHING;
