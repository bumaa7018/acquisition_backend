-- `v_parcel_status` харагдацад ӨНГӨНИЙ баганыг баталгаажуулна.
--
-- ЯАГААД ТУСДАА MIGRATION: 000033 нь эхэндээ `color` БАГАНАГҮЙ бичигдэж,
-- зарим орчинд тэр хувилбараараа аль хэдийн хэрэгжсэн. golang-migrate нь
-- хэрэгжсэн хувилбарыг дахин ажиллуулдаггүй тул 000033-ын файлыг засахад
-- тэдгээр орчны харагдац ХУУЧНААРАА үлдэнэ.
--
-- ҮР ДАГАВАР нь чимээгүй биш, БҮТЭН БУУЛТ байсан: `parcel_status.sld` загвар
-- өнгийг `<PropertyName>color</PropertyName>`-ээр уншдаг тул багана байхгүй
-- үед GeoServer зураг БИШ, алдаа буцаана:
--
--   ServiceException: The requested Style can not be used with this layer.
--   The style specifies an attribute named 'color', not found in the layer
--
-- Ингэснээр газрын зураг дээр нэгж талбар ОГТ харагдахгүй болно.
--
-- CREATE OR REPLACE ашиглах боломжгүй: `color` нь баганы ДУНД (geometry-ийн
-- өмнө) нэмэгддэг тул PostgreSQL татгалзана. Иймд DROP + CREATE хийнэ.
-- Харагдац устгагдаад шууд үүсэх тул GeoServer-ийн давхаргад хөндлөнгийн
-- тохиргоо шаардлагагүй (нэрлэлт, багануудын нэр хэвээр).
DROP VIEW IF EXISTS public.v_parcel_status;
CREATE VIEW public.v_parcel_status AS
SELECT p.id,
       p.parcel_id,
       p.acquisition_id,
       p.acquisition_area_m2,
       p.status,
       -- Өнгө нь `parcel_status` БҮРТГЭЛЭЭС. COALESCE нь өгөгдмөл саарлыг энд
       -- барина: өнгө тохируулаагүй төлөв SLD дээр хоосон утга болж
       -- зурагдахгүй үлдэхээс сэргийлнэ.
       COALESCE(ps.color, '#94a3b8') AS color,
       p.acquisition_geom::geometry(Polygon, 4326) AS geometry,
       EXTRACT(YEAR FROM (
           SELECT MAX(psh.status_date) FROM public.parcel_status_history psh
           WHERE psh.parcel_id = p.parcel_id
       ))::INTEGER AS status_year,
       COALESCE(',' || (SELECT string_agg(laa.user_id::text, ',')
           FROM public.land_acquisition_assignee laa
           WHERE laa.acquisition_id = p.acquisition_id) || ',', '') AS assignee_user_ids
FROM public.parcel p
LEFT JOIN public.parcel_status ps ON ps.id = p.status
WHERE p.acquisition_geom IS NOT NULL;
