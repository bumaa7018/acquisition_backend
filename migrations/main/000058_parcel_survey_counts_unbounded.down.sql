-- 3-аас дээшх утгыг «3+» (3) болгож, хуучин 1..3 хязгаарыг сэргээнэ.
UPDATE public.parcel_survey SET
    chimney_count    = LEAST(chimney_count, 3),
    toilet_count     = LEAST(toilet_count, 3),
    cohabitant_count = LEAST(cohabitant_count, 3)
WHERE chimney_count > 3 OR toilet_count > 3 OR cohabitant_count > 3;

ALTER TABLE public.parcel_survey DROP CONSTRAINT IF EXISTS parcel_survey_counts_check;
ALTER TABLE public.parcel_survey
    ADD CONSTRAINT parcel_survey_counts_check CHECK (
        (chimney_count IS NULL OR chimney_count BETWEEN 1 AND 3)
        AND (toilet_count IS NULL OR toilet_count BETWEEN 1 AND 3)
        AND (cohabitant_count IS NULL OR cohabitant_count BETWEEN 1 AND 3)
    );
