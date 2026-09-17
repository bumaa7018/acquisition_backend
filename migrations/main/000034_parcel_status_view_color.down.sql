-- Буцаалт: өнгөний баганагүй хувилбар руу (000033-ын анхны хэлбэр).
-- АНХААР: буцаасны дараа `parcel_status.sld` загвар ажиллахаа болино —
-- хамт нь хуучин `parcel_s0..s5` загваруудад шилжүүлэх хэрэгтэй.
DROP VIEW IF EXISTS public.v_parcel_status;
CREATE VIEW public.v_parcel_status AS
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
WHERE p.acquisition_geom IS NOT NULL;
