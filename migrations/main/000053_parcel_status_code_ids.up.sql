-- Нэгж талбарын төлөвийн id нь code-той ИЖИЛ болно (0–8 → 101–109). Өгөгдлийн
-- шилжүүлэг (parcel.status, түүх, ажлын урсгал) нь seeder-ийн
-- 000053_parcel_status_code_ids алхамд — энд анхны төлөвийн DEFAULT ба
-- төлөвийн id-аар хатуу шүүдэг хуучин GeoServer харагдацууд.
ALTER TABLE public.parcel ALTER COLUMN status SET DEFAULT 101;

-- v_parcel_s0..s5 (000027) — давхаргын нэр хуучин id-аар (s0 = 0 …) тул
-- нэрийг хэвээр үлдээж, шүүлтийг шинэ id-аар:
--   s0 → 101 нөлөөлөлд өртсөн   s1 → 105 зөвшилцөх   s2 → 104 үнэлгээ
--   s3 → 103 нөлөөллөөс гаргасан s4 → 106 татгалзсан s5 → 109 чөлөөлсөн
-- Багана хэвээр тул CREATE OR REPLACE хангалттай.
DO $$
DECLARE
    m int[][] := ARRAY[[0, 101], [1, 105], [2, 104], [3, 103], [4, 106], [5, 109]];
    i int;
BEGIN
    FOR i IN 1 .. array_length(m, 1) LOOP
        EXECUTE format($v$
CREATE OR REPLACE VIEW public.v_parcel_s%s AS
SELECT p.id,
       p.parcel_id,
       p.acquisition_id,
       p.acquisition_area_m2,
       p.status,
       p.acquisition_geom::geometry(Polygon, 4326) AS geometry,
       EXTRACT(YEAR FROM (
           SELECT MAX(psh.status_date) FROM public.parcel_status_history psh
           WHERE psh.parcel_id = p.parcel_id
       ))::INTEGER AS status_year,
       COALESCE(',' || (SELECT string_agg(laa.user_id::text, ',')
           FROM public.land_acquisition_assignee laa
           WHERE laa.acquisition_id = p.acquisition_id) || ',', '') AS assignee_user_ids
FROM public.parcel p
WHERE p.acquisition_geom IS NOT NULL AND p.status = %s$v$, m[i][1], m[i][2]);
    END LOOP;
END $$;
