-- GEOSERVER-ИЙН НИЙТЛЭГДЭХ VIEW-УУД.
--
-- Асуудал: эдгээр view нь урьд нь ЗӨВХӨН `government-geoserver/Makefile`-ийн
-- `config` таргетын [1/4] алхамд үүсдэг байсан. `make reset-db` нь appdb-г
-- бүтнээр нь DROP хийж дахин үүсгэдэг тул reset хийсний дараа view-ууд
-- АЛГА болж, GeoServer-ийн давхаргууд (land:v_parcel_*, land:v_acquisition_*)
-- "relation does not exist"-ээр унадаг — газрын зураг хоосон харагдана.
-- Сэргээхийн тулд өөр репог (government-geoserver) гараар ажиллуулах шаардлагатай
-- байв.
--
-- Иймд view-ийн тодорхойлолтыг ЭНД, миграцад авчирлаа: `make reset-db` бол
-- миграцийг API асахад ажиллуулдаг тул view-ууд АВТОМАТААР дахин үүснэ.
-- `government-geoserver`-ийн [1/4] алхам нь ЯГ ижил тодорхойлолттой ба
-- DROP+CREATE хийдэг тул хоёр талаас ажиллуулахад зөрчилдөхгүй (idempotent).
-- Тодорхойлолт өөрчлөгдвөл ХОЁУЛАНГ нь зэрэг засна.
--
-- АНХААР (ирээдүйн миграц бичигчид):
-- View нь доорх хүснэгтүүдээс ХАМААРАЛТАЙ тул тэдгээрийн багана дээр
-- `ALTER COLUMN ... TYPE` / `DROP COLUMN` хийх миграц нь "cannot alter type of
-- a column used by a view" гэж УНАНА. Тийм миграц бичих бол эхэлж холбогдох
-- view-г DROP хийгээд, төгсгөлд нь энэ файлын тодорхойлолтоор дахин үүсгэнэ.
-- (Багана НЭМЭХ `ADD COLUMN` нь view-д огт нөлөөлөхгүй — чөлөөтэй.)

-- ЧӨЛӨӨЛӨЛТИЙН ХИЛ (баталсан хил, geometry).
DROP VIEW IF EXISTS public.v_acquisition_boundary;
CREATE VIEW public.v_acquisition_boundary AS
SELECT id AS acquisition_id,
       plan_code,
       COALESCE(acquisition_name, '') AS acquisition_name,
       status,
       start_date,
       end_date,
       area_m2,
       geometry::geometry(Polygon, 4326) AS geometry
FROM public.land_acquisition
WHERE geometry IS NOT NULL;

-- ЧӨЛӨӨЛӨЛТИЙН ТӨЛӨВЛӨГӨӨНИЙ хил (plan_geom).
DROP VIEW IF EXISTS public.v_acquisition_plan;
CREATE VIEW public.v_acquisition_plan AS
SELECT id AS acquisition_id,
       plan_code,
       COALESCE(acquisition_name, '') AS acquisition_name,
       status,
       start_date,
       end_date,
       plan_area_m2,
       plan_geom::geometry(Polygon, 4326) AS geometry
FROM public.land_acquisition
WHERE plan_geom IS NOT NULL;

-- v_acquisition_boundary-тэй ижил, харин УСТГАСАН (soft delete) чөлөөлөлтийг
-- хасна. Төлөвлөгөөний дэлгэц үүнийг ашиглана.
DROP VIEW IF EXISTS public.v_plan_acquisition;
CREATE VIEW public.v_plan_acquisition AS
SELECT id AS acquisition_id,
       plan_code,
       COALESCE(acquisition_name, '') AS acquisition_name,
       status,
       start_date,
       end_date,
       area_m2,
       geometry::geometry(Polygon, 4326) AS geometry
FROM public.land_acquisition
WHERE geometry IS NOT NULL
  AND deleted_at IS NULL;

-- ЧӨЛӨӨЛӨЛТӨД ӨРТСӨН нэгж талбарууд (давхцлын геометрээр).
DROP VIEW IF EXISTS public.v_parcel_acquisition;
CREATE VIEW public.v_parcel_acquisition AS
SELECT p.id,
       p.parcel_id,
       p.acquisition_id,
       p.acquisition_area_m2,
       p.status,
       p.acquisition_geom::geometry(Polygon, 4326) AS geometry
FROM public.parcel p
WHERE p.acquisition_geom IS NOT NULL;

-- ЯВЦААР (төлөвөөр) салгасан давхаргууд: 0..5.
--
-- Тус бүр нь тусдаа давхарга болж нийтлэгддэг — frontend нь явцыг нь
-- давхаргаар асаах/унтраах тул нэг view дээр CQL шүүхээс хурдан.
--
--   status_year        — тухайн төлөвт хамгийн сүүлд шилжсэн ОН (шүүлтэд).
--   assignee_user_ids  — чөлөөлөлтийн хариуцагчдын ID, `,1,7,` хэлбэрээр.
--                        GeoServer дээр массив дамжуулах боломжгүй тул
--                        `CQL_FILTER=assignee_user_ids LIKE '%,7,%'` гэж
--                        шүүхээр таслалаар хүрээлж хадгална.
DROP VIEW IF EXISTS public.v_parcel_s0;
CREATE VIEW public.v_parcel_s0 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 0;

DROP VIEW IF EXISTS public.v_parcel_s1;
CREATE VIEW public.v_parcel_s1 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 1;

DROP VIEW IF EXISTS public.v_parcel_s2;
CREATE VIEW public.v_parcel_s2 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 2;

DROP VIEW IF EXISTS public.v_parcel_s3;
CREATE VIEW public.v_parcel_s3 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 3;

DROP VIEW IF EXISTS public.v_parcel_s4;
CREATE VIEW public.v_parcel_s4 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 4;

DROP VIEW IF EXISTS public.v_parcel_s5;
CREATE VIEW public.v_parcel_s5 AS
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
WHERE p.acquisition_geom IS NOT NULL AND p.status = 5;

-- НЭЭЛТТЭЙ давхарга (эрх шалгахгүй, гадны системд).
--
-- docs/public-map-layer.md-г үз. Эзэмшигчийн нэр/холбоо барих, нөхөх олговрын
-- дүн зэрэг нь ЭНД ОГТ БАЙХГҮЙ — зөвхөн байршил, засаг захиргааны нэгж, явц.
-- Давхцлын геометр байхгүй бол нэгж талбарын үндсэн хилээр орлуулна.
DROP VIEW IF EXISTS public.v_parcel_public;
CREATE VIEW public.v_parcel_public AS
SELECT p.id,
       p.parcel_id,
       p.au1_code,
       p.au2_code,
       p.au3_code,
       COALESCE(a1.name, '') AS au1_name,
       COALESCE(a2.name, '') AS au2_name,
       COALESCE(a3.name, '') AS au3_name,
       p.status,
       COALESCE(ps.name, '') AS status_name,
       COALESCE(p.acquisition_area_m2, 0) AS acquisition_area_m2,
       COALESCE(p.acquisition_geom, p.geometry)::geometry(Polygon, 4326) AS geometry
FROM public.parcel p
JOIN public.land_acquisition la ON la.id = p.acquisition_id AND la.deleted_at IS NULL
LEFT JOIN public.parcel_status ps ON ps.id = p.status
LEFT JOIN public.au1 a1 ON a1.code = p.au1_code
LEFT JOIN public.au2 a2 ON a2.code = p.au2_code
LEFT JOIN public.au3 a3 ON a3.code = p.au3_code
WHERE COALESCE(p.acquisition_geom, p.geometry) IS NOT NULL;
