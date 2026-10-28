-- Судалгааны тоон хариулт хязгааргүй болно (маягтад «+»-ээр нэмдэг).
-- Өмнө нь 1..3 (3 = «3+») байсан; дээд хязгаар нь зөвхөн алдаатай орцоос
-- хамгаална (domain.SurveyCountMax). Статистик 3-аас дээшхийг «3+»-д нэгтгэнэ.

ALTER TABLE public.parcel_survey DROP CONSTRAINT IF EXISTS parcel_survey_counts_check;
ALTER TABLE public.parcel_survey
    ADD CONSTRAINT parcel_survey_counts_check CHECK (
        (chimney_count IS NULL OR chimney_count BETWEEN 1 AND 999)
        AND (toilet_count IS NULL OR toilet_count BETWEEN 1 AND 999)
        AND (cohabitant_count IS NULL OR cohabitant_count BETWEEN 1 AND 999)
    );
