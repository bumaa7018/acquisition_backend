-- Нэгж талбарын төлөвийн НЭГДСЭН GeoServer харагдац.
--
-- ЯАГААД: өмнө нь төлөв бүрд ТУСДАА харагдац (v_parcel_s0 … v_parcel_s5),
-- тусдаа SLD (parcel_s0.sld …), GeoServer-ийн каталогт тусдаа давхарга, мөн
-- frontend/proxy дээр тус тусдаа бүртгэл байсан. Бүртгэлд ШИНЭ төлөв нэмэхэд
-- эдгээр бүгдийг ГАРААР нэмэх шаардлагатай байв — тиймээс "төлөв нэмэх"
-- үйлдэл нь давхарга дээр хэзээ ч тусдаггүй байлаа.
--
-- Энэ харагдац нь БҮХ төлвийг агуулна. Тодорхой нэг төлвийг GeoServer-ийн
-- `CQL_FILTER=status=N` -ээр ялгана.
--
-- ӨНГӨ нь ЭНЭ ХАРАГДАЦААС гарна (`color` багана) — SLD түүнийг
-- `<PropertyName>color</PropertyName>`-ээр шууд уншина. Хүсэлтээр
-- дамжуулахгүй: өнгө нь `parcel_status` бүртгэлийн шинж чанар тул түүнийг
-- дуудагч тал бүр давтан илгээх ёсгүй, мөн дур мэдэн өөрчлөх ч ёсгүй.
-- Ингэснээр төлөв нэмэх/өнгө солиход GeoServer-т ч, frontend-д ч ямар ч
-- өөрчлөлт хийх шаардлагагүй — зөвхөн бүртгэл засна.
--
-- COALESCE нь ӨГӨГДМӨЛ саарал өгнө: өнгө тохируулаагүй (NULL) төлөв SLD дээр
-- хоосон утга болж зурагдахгүй үлдэхээс сэргийлнэ. Өгөгдмөлийг ЭНД барих нь
-- SLD дотор NULL шалгах дүрэм бичихээс хялбар бөгөөд хэвлэх загвартай ч
-- автоматаар нийцнэ.
--
-- Бусад баганууд нь v_parcel_s0..s5-тай ЯГ ижил (`status_year`,
-- `assignee_user_ids` зэрэг нь frontend-ийн одоогийн шүүлтүүдэд хэрэглэгддэг).
DROP VIEW IF EXISTS public.v_parcel_status;
CREATE VIEW public.v_parcel_status AS
SELECT p.id,
       p.parcel_id,
       p.acquisition_id,
       p.acquisition_area_m2,
       p.status,
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

-- ХУУЧИН v_parcel_s0..s5-ийг ЭНД УСТГАХГҮЙ. Шалтгаан: тэдгээр нь GeoServer-ийн
-- каталогт давхарга болж бүртгэлтэй бөгөөд харагдацыг нь урьдчилан устгавал
-- GeoServer тэр давхаргууд дээр алдаа өгнө. Frontend шинэ давхаргад бүрэн
-- шилжиж, GeoServer-ийн каталогоос хуучин давхаргууд хасагдсаны ДАРАА тусдаа
-- migration-аар устгана.
