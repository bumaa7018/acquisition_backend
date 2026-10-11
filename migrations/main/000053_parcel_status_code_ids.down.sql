ALTER TABLE public.parcel ALTER COLUMN status SET DEFAULT 0;

-- v_parcel_s0..s5 — 000027-ийн шүүлт (s<N> = status N).
DO $$
DECLARE
    i int;
BEGIN
    FOR i IN 0 .. 5 LOOP
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = %s$v$, i, i);
    END LOOP;
END $$;
