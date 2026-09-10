ALTER TABLE public.parcel
    ADD COLUMN IF NOT EXISTS estimated_confidence_percent double precision;

COMMENT ON COLUMN public.parcel.estimated_confidence_percent IS
    'Төсөөллийн үнэлгээний бүх бүсэд ижил хэрэглэх итгэлцүүр (%). ГУС-ийн төлбөрийн итгэлцүүрийг өөрчлөхгүй.';
