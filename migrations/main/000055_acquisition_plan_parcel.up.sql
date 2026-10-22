-- Чөлөөлөлтийн ТӨЛӨВЛӨГӨӨНИЙ НЭГЖ ТАЛБАР ба түүн дээрх бүтээн байгуулалт
-- (gazner). plan хүснэгт нь plan_code-оор (төслийн код, ж: 2026-04-011-9878)
-- ЦОРЫН ГАНЦ мөртэй боловч нэг төсөлд олон төлөвлөгөөний нэгж талбар, тус бүр
-- өөрийн gazner-тэй байдаг — plan.parcel_id/gazner нь тэдгээрийн аль НЭГнийх
-- тул бүх чөлөөлөлт дээр ижил (буруу) бүтээн байгуулалт харагдаж байв.
-- Өмнө үүссэн чөлөөлөлтийн утгыг гараар нөхнө (seeder байхгүй).
ALTER TABLE public.land_acquisition
    ADD COLUMN IF NOT EXISTS plan_parcel_id text,
    ADD COLUMN IF NOT EXISTS plan_gazner    text;

CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_parcel_id
    ON public.land_acquisition (plan_parcel_id);
