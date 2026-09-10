ALTER TABLE public.parcel
    ADD COLUMN IF NOT EXISTS estimated_base_fee_per_m2 double precision;

COMMENT ON COLUMN public.parcel.estimated_base_fee_per_m2 IS
    'Газрын төлбөрийн бодолт байхгүй үед төсөөллийн үнэлгээнд гараас оруулсан суурь төлбөр (₮/м²).';
