-- ТАЛБАЙ ТООЦОХ АРГЫГ ГУС-ТАЙ НЭГТГЭХ.
--
-- Асуудал: бид чөлөөлөх талбайг `ST_Area(geom::geography)`-аар (WGS84
-- эллипсоид дээрх геодезийн талбай) боддог байсан бол ГУС нь UTM проекц дээр
-- хавтгайгаар боддог. UTM-д k0 = 0.9996 тул төв меридиан дээр талбай ~0.08%
-- бага, бүсийн зах руу очих тусам өснө. Улмаар нэг л полигон дээр хоёр өөр
-- тоо гарч, `remaining_area_m2 = area_m2 - acquisition_area_m2` нь ГУС-ийн
-- (UTM) нийт талбайгаас бидний (геодезийн) талбайг хасах болж байсан.
--
-- Шийдэл: ГУС-ийн base.calculate_area_utm / base.find_utm_srid-ийн ЯГ ИЖИЛ
-- хуулбарыг public схемд үүсгэж, бүх талбайн тооцоог үүгээр гүйцэтгэнэ.
--
-- ЭХ ФУНКЦЭЭС ЗАРИМ ЗҮЙЛИЙГ ЗОРИУД ХЭВЭЭР ҮЛДЭЭВ (зөрөхгүйн тулд):
--   * бүсийг ST_Centroid биш ST_PointOnSurface-оор олно (полигон дотор
--     баталгаатай байх цэг);
--   * `round(...)` нь аравтын орон ЗААГААГҮЙ — бүхэл м² болж бөөрөнхийлнө;
--   * 180-р уртраг дээрх тусгай тохиолдол (base + 60).

-- ТӨРЛИЙГ ЗААВАЛ `public.geometry` гэж бүтнээр бичнэ (зүгээр `geometry` БИШ).
--
-- Миграцууд НЭГ холболт дээр дараалан ажилладаг ба postgis extension нь яг тэр
-- холболтын дотор, 000001 baseline-д шинээр үүсдэг. Тэр session-ий search_path
-- дагуух төрлийн хайлт нь extension үүсэхээс өмнөх төлөвөөр кэшлэгдсэн байдаг
-- тул схемгүй `geometry` нь "type geometry does not exist" гэж унана (цоо шинэ
-- санд `make reset-db` хийхэд ЯГ энэ алдаа гарч, сан dirty болж байсан).
-- Схемээр бүтэн заасан нэр search_path-аас хамаарахгүй тул найдвартай.
CREATE OR REPLACE FUNCTION public.find_utm_srid(p public.geometry)
RETURNS integer
LANGUAGE plpgsql
STABLE
AS $function$
DECLARE
    out_srid  integer;
    base_srid integer;
BEGIN
    IF public.ST_SRID(p) != 4326 THEN
        RAISE NOTICE 'find_utm_srid: input geometry has wrong SRID (%)', public.ST_SRID(p);
        RETURN NULL;
    END IF;

    IF public.ST_Y(p) < 0 THEN
        --- south hemisphere
        base_srid := 32700;
    ELSE
        --- north hemisphere or on equator
        base_srid := 32600;
    END IF;

    out_srid := base_srid + floor((public.ST_X(p) + 186) / 6);
    IF (public.ST_X(p) = 180) THEN
        out_srid := base_srid + 60;
    END IF;

    RETURN out_srid;
END;
$function$;

CREATE OR REPLACE FUNCTION public.calculate_area_utm(public.geometry)
RETURNS double precision
LANGUAGE plpgsql
STABLE
AS $function$
DECLARE
    v_inputgeo alias for $1;
    v_utmsrid  integer;
    v_area     double precision;
BEGIN
    IF v_inputgeo IS NULL OR public.ST_IsEmpty(v_inputgeo) THEN
        RETURN NULL;
    END IF;

    SELECT public.find_utm_srid(public.ST_PointOnSurface(public.ST_MakeValid(v_inputgeo)))
      INTO v_utmsrid;
    IF v_utmsrid IS NULL THEN
        RETURN NULL;
    END IF;

    v_area := round(public.ST_Area(
        public.ST_Transform(public.ST_MakeValid(v_inputgeo), v_utmsrid)
    )::numeric);

    RETURN v_area;
END;
$function$;

COMMENT ON FUNCTION public.find_utm_srid(public.geometry) IS
    'ГУС-ийн base.find_utm_srid-ийн хуулбар — 4326 цэгээс UTM SRID (326xx/327xx).';
COMMENT ON FUNCTION public.calculate_area_utm(public.geometry) IS
    'ГУС-ийн base.calculate_area_utm-ийн хуулбар — UTM проекц дээрх талбай, бүхэл м². Талбайн тооцоог ГУС-тай нэг мөр байлгана.';
