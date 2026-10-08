-- Төсөл хянах нэгж — захирамжийн төслийн «Хянагдаж буй» явцын «Хянагдаж буй
-- газар» сонголт (өмнө нь «Хэнд» чөлөөт текст байсан). Тохиргооноос удирдана.
CREATE TABLE IF NOT EXISTS public.decision_review_unit (
    id         serial PRIMARY KEY,
    code       varchar(50)  NOT NULL UNIQUE,
    name       varchar(200) NOT NULL,
    sort_order integer      NOT NULL DEFAULT 0
);

-- Явцын бичлэг сонгосон нэгжтэй холбогдоно. recipient нь нэгжийн НЭРИЙН
-- хуулбар хэвээр (түүх, тайлан). Холбогдсон нэгжийг устгахгүй (FK).
ALTER TABLE public.decision_draft_progress_history
    ADD COLUMN IF NOT EXISTS review_unit_id integer REFERENCES public.decision_review_unit(id);
