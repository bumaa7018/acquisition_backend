# Нээлттэй газрын зургийн давхарга (гадны системд)

Төлөвлөгөөний **2.9.2** ажил: бусад системд өгөгдөл холбож харуулах.

`land:v_parcel_public` — **чөлөөлөлтөд өртсөн БҮХ нэгж талбар**, явцын
(төлөвийн) нэр ба өнгөтэйгээр. **Эрх шалгахгүй** (нэвтрэлт шаардахгүй) уншина.

## Хаяг

```
WMS : https://<host>/api/public/geoserver/land/wms
WFS : https://<host>/api/public/geoserver/land/ows
```

GeoServer нь өөрөө зөвхөн `127.0.0.1`-д нийтлэгддэг тул бүх хандалт дээрх
гарцаар явна. Гарц нь `Access-Control-Allow-Origin: *` буцаадаг — хөтөч дээрх
клиентээс шууд дуудаж болно.

## Давхаргын атрибутууд

| Талбар | Тайлбар |
|---|---|
| `parcel_id` | Нэгж талбарын дугаар |
| `au1_code` / `au1_name` | Аймаг, нийслэл |
| `au2_code` / `au2_name` | **Сум, дүүрэг** (шүүлтэд) |
| `au3_code` / `au3_name` | Баг, хороо |
| `status` | Явцын код (0-5) |
| `status_name` | **Явцын нэр** (бичээсээр газрын зурагт харагдана) |
| `acquisition_area_m2` | Чөлөөлөгдсөн (өртсөн) талбай, м² |
| `geometry` | Полигон (EPSG:4326). Давхцлын геометр байхгүй бол нэгж талбарын үндсэн хил |

Эзэмшигчийн нэр/холбоо барих мэдээлэл, нөхөх олговрын дүн, чөлөөлөлтийн дотоод
ID зэрэг нь давхаргад **байхгүй**.

## Явцын өнгө

| Код | Явц | Өнгө |
|---|---|---|
| 0 | Хүлээгдэж буй | `#64748b` саарал |
| 1 | Зөвшилцөх шатандаа | `#facc15` шар |
| 2 | Үнэлгээ хийх | `#f97316` улбар шар |
| 3 | Нөлөөлөгдсөн гарсан | `#3b82f6` цэнхэр |
| 4 | Татгалзсан | `#ef4444` улаан |
| 5 | Чөлөөлсөн | `#22c55e` ногоон |

Өнгө нь системийн дэлгэцтэй ижил (`parcel_public.sld`). Өнгөний тайлбарыг
зурган хэлбэрээр `GetLegendGraphic`-аар авна.

## Жишээ дуудалтууд

```bash
BASE=https://<host>/api/public/geoserver

# 1. Зураг (WMS GetMap) — Улаанбаатар орчим
curl "$BASE/land/wms?service=WMS&version=1.3.0&request=GetMap\
&layers=land:v_parcel_public&styles=&crs=EPSG:4326\
&bbox=47.85,106.70,47.95,106.95&width=900&height=700\
&format=image/png&transparent=true" -o parcels.png

# 2. ДҮҮРГЭЭР шүүх (Баянгол = 01107)
curl "$BASE/land/wms?...&CQL_FILTER=au2_code='01107'" -o bayangol.png

# 3. Явцаар шүүх (Чөлөөлсөн)
curl "$BASE/land/wms?...&CQL_FILTER=status=5" -o released.png

# 4. Өгөгдөл (WFS GeoJSON) — дүүргээр
curl "$BASE/land/ows?service=WFS&version=2.0.0&request=GetFeature\
&typeNames=land:v_parcel_public&outputFormat=application/json\
&count=1000&CQL_FILTER=au2_code='01107'"

# 5. Өнгөний тайлбар (легенд)
curl "$BASE/land/wms?service=WMS&version=1.3.0&request=GetLegendGraphic\
&layer=land:v_parcel_public&format=image/png" -o legend.png
```

## Гадны байгууллагад ямар линк өгөх вэ

WMS/WFS дуудаж чаддаг систем (QGIS, ArcGIS, Leaflet, OpenLayers, MapServer,
өөр GeoServer) дараах хаягийг **шууд** тохируулж ажиллуулна:

```
WMS: https://<host>/api/public/geoserver/land/wms
WFS: https://<host>/api/public/geoserver/land/ows
```

- QGIS: *Layer → Add Layer → Add WMS/WMTS Layer* → New → URL-д WMS хаягийг
  тавина → Connect → `v_parcel_public`. (WFS-д мөн адил.)
- `GetCapabilities` ажиллана. Гарц нь GeoServer-ийн хариултыг ЗАСААД буцаана:
  дотоод хаяг (`localhost:8600`) гарцын бүтэн хаягаар солигдож, зөвхөн
  `v_parcel_public` давхарга жагсагдана. Иймд клиент дараагийн хүсэлтээ
  зөв хаягаар явуулна.
- Хаяг нь хүсэлтийн `Host` / `X-Forwarded-Host` толгойгоос тодорхойлогддог тул
  reverse proxy-ийн ард нэмэлт тохиргоо ШААРДАХГҮЙ.

**ЗААВАЛ:** frontend (Next) сервер нь тэдгээр системээс СҮЛЖЭЭГЭЭР хүрэхүйц
байх ёстой. GeoServer өөрөө `127.0.0.1`-д хаалттай тул `:8600` порт руу шууд
дуудах боломжгүй — гадны систем зөвхөн дээрх гарцаар л хандана.

## Хязгаарлалт (гарц дээр хатуу хэрэгжсэн)

- Зам: зөвхөн `land/wms`, `land/ows`, `land/wfs`. `rest/*` (админ REST),
  `web/*` (админ UI), `gwc/*` (tile гарц) → **403**.
- Метод: `GET`, `HEAD`, `OPTIONS`. Бусад → 405.
- Үйлдэл: `GetMap`, `GetFeatureInfo`, `GetLegendGraphic`, `GetCapabilities`,
  `GetFeature`, `DescribeFeatureType`. Бусад (ж: WFS `Transaction` = бичих)
  → **403**.
- **Давхарга**: `LAYERS` / `typeNames` -ийг сервер өөрөө `v_parcel_public`
  болгож дардаг — өөр давхаргын өгөгдөл гарахгүй (`land:parcel` гэж дуудсан ч
  нээлттэй давхарга буцна).
- `STYLES` дарагдана — өөр style дуудах боломжгүй.
- WFS-ээр нэг хүсэлтэд дээд тал нь **5000 мөр**.

## Дахин тохируулах

Давхарга, view, style нь GeoServer-ийн тохиргоонд байна:

```bash
cd government-geoserver && make config
```

- View: `v_parcel_public` (Makefile-ийн [1/4] алхам)
- Style: `styles/parcel_public.sld` → `land:parcel_public`
- Давхарга: `land:v_parcel_public`
