DROP TRIGGER IF EXISTS trg_parcel_survey_auto_create ON public.parcel;
DROP FUNCTION IF EXISTS public.parcel_survey_auto_create();

-- Автоматаар үүссэн, хариултгүй хоосон бүртгэлүүдийг цэвэрлэнэ.
DELETE FROM public.parcel_survey
WHERE chimney_count IS NULL AND toilet_count IS NULL AND cohabitant_count IS NULL
  AND relocation_type IS NULL AND relocation_address IS NULL AND note IS NULL;
