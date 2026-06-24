-- ============================================================
-- Geometry нөхөлт: seed_mock_data.sql-р үүссэн өгөгдөлд
-- Ажиллуулах: psql "$DATABASE_URL" -f scripts/patch_mock_geometry.sql
-- ============================================================
--
-- Парцел ID формат: "DD-KK-BB-PPP"
--   DD  = дүүрэг (11-14)
--   KK  = хороо  (01-10)
--   BB  = блок   (01-10)
--   PPP = парцел (001-005)
--
-- Дүүрэг тус бүрийн grid (WGS84 / EPSG:4326):
--   11 Хан-Уул:        base_lon=106.820  base_lat=47.870  Δlon_block=0.016  Δlat_khoroo=0.009
--   12 Баянзүрх:       base_lon=107.010  base_lat=47.890  Δlon_block=0.019  Δlat_khoroo=0.013
--   13 Сонгинохайрхан: base_lon=106.650  base_lat=47.890  Δlon_block=0.017  Δlat_khoroo=0.021
--   14 Чингэлтэй:      base_lon=106.870  base_lat=47.900  Δlon_block=0.014  Δlat_khoroo=0.008
-- ============================================================

-- ── 1. Нэгж талбарын geometry ─────────────────────────────────────────────
-- parcel_id-аас дүүрэг/хороо/блок/парцел индекс гаргаж POLYGON тооцно.
-- Блок бүрийг 5 тэнцүү зурвасаар (lon дагуу) хувааж парцел авна.

UPDATE parcel p
SET geometry = ST_SetSRID(
  ST_MakePolygon(ST_MakeLine(ARRAY[
    ST_MakePoint(c.x0, c.y0),
    ST_MakePoint(c.x1, c.y0),
    ST_MakePoint(c.x1, c.y1),
    ST_MakePoint(c.x0, c.y1),
    ST_MakePoint(c.x0, c.y0)
  ])), 4326)::geometry(Polygon, 4326)
FROM (
  SELECT
    id,
    split_part(parcel_id, '-', 1)::int AS d,
    split_part(parcel_id, '-', 2)::int AS kh,
    split_part(parcel_id, '-', 3)::int AS bl,
    split_part(parcel_id, '-', 4)::int AS pn
  FROM parcel
  WHERE parcel_id ~ '^(11|12|13|14)-'
    AND geometry IS NULL
) q
CROSS JOIN LATERAL (
  SELECT
    -- lon_min = base + (block-1)*block_step + (parcel-1)*parcel_step
    CASE q.d
      WHEN 11 THEN 106.820 + (q.bl - 1) * 0.016 + (q.pn - 1) * 0.0032
      WHEN 12 THEN 107.010 + (q.bl - 1) * 0.019 + (q.pn - 1) * 0.0038
      WHEN 13 THEN 106.650 + (q.bl - 1) * 0.017 + (q.pn - 1) * 0.0034
      WHEN 14 THEN 106.870 + (q.bl - 1) * 0.014 + (q.pn - 1) * 0.0028
    END AS x0,
    -- lon_max = lon_min + parcel_step
    CASE q.d
      WHEN 11 THEN 106.820 + (q.bl - 1) * 0.016 + q.pn * 0.0032
      WHEN 12 THEN 107.010 + (q.bl - 1) * 0.019 + q.pn * 0.0038
      WHEN 13 THEN 106.650 + (q.bl - 1) * 0.017 + q.pn * 0.0034
      WHEN 14 THEN 106.870 + (q.bl - 1) * 0.014 + q.pn * 0.0028
    END AS x1,
    -- lat_min = base + (khoroo-1)*khoroo_step
    CASE q.d
      WHEN 11 THEN 47.870 + (q.kh - 1) * 0.009
      WHEN 12 THEN 47.890 + (q.kh - 1) * 0.013
      WHEN 13 THEN 47.890 + (q.kh - 1) * 0.021
      WHEN 14 THEN 47.900 + (q.kh - 1) * 0.008
    END AS y0,
    -- lat_max = lat_min + khoroo_step
    CASE q.d
      WHEN 11 THEN 47.870 + q.kh * 0.009
      WHEN 12 THEN 47.890 + q.kh * 0.013
      WHEN 13 THEN 47.890 + q.kh * 0.021
      WHEN 14 THEN 47.900 + q.kh * 0.008
    END AS y1
) c
WHERE p.id = q.id;

-- ── 2. Land acquisition geometry ──────────────────────────────────────────
-- Чөлөөлөлт тус бүрт acq_names-д дурдсан дүүрэг/хороонд тохирсон polygon.
-- NULL байвал л шинэчилнэ; аль хэдийн утгатай мөрийг хөндөхгүй.

UPDATE land_acquisition la
SET
  geometry  = v.geom,
  plan_geom = CASE WHEN la.plan_geom IS NULL THEN v.geom ELSE la.plan_geom END
FROM (VALUES
  -- Хан-Уул дүүрэг
  ('ЗТ-2023-001', ST_MakeEnvelope(106.900, 47.930, 106.940, 47.960, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2024-031', ST_MakeEnvelope(106.840, 47.880, 106.890, 47.920, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2022-034', ST_MakeEnvelope(106.860, 47.900, 106.920, 47.940, 4326)::geometry(Polygon,4326)),
  -- Баянзүрх дүүрэг
  ('ЗТ-2023-015', ST_MakeEnvelope(107.030, 47.900, 107.090, 47.940, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2023-042', ST_MakeEnvelope(107.070, 47.930, 107.130, 47.980, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2023-071', ST_MakeEnvelope(107.100, 47.950, 107.160, 48.000, 4326)::geometry(Polygon,4326)),
  -- Сонгинохайрхан дүүрэг
  ('ЗТ-2024-003', ST_MakeEnvelope(106.660, 47.900, 106.730, 47.960, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2021-005', ST_MakeEnvelope(106.740, 48.000, 106.820, 48.080, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2022-088', ST_MakeEnvelope(106.700, 47.940, 106.770, 48.000, 4326)::geometry(Polygon,4326)),
  -- Чингэлтэй дүүрэг
  ('ЗТ-2024-022', ST_MakeEnvelope(106.880, 47.900, 106.940, 47.940, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2025-002', ST_MakeEnvelope(106.900, 47.930, 106.960, 47.970, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2024-099', ST_MakeEnvelope(106.920, 47.940, 106.980, 47.980, 4326)::geometry(Polygon,4326)),
  -- Сүхбаатар дүүрэг
  ('ЗТ-2024-007', ST_MakeEnvelope(106.970, 47.910, 107.020, 47.950, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2024-018', ST_MakeEnvelope(106.980, 47.910, 107.030, 47.950, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2023-112', ST_MakeEnvelope(107.000, 47.920, 107.050, 47.960, 4326)::geometry(Polygon,4326)),
  -- Баянгол дүүрэг
  ('ЗТ-2023-008', ST_MakeEnvelope(106.850, 47.890, 106.910, 47.930, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2023-027', ST_MakeEnvelope(106.870, 47.890, 106.930, 47.930, 4326)::geometry(Polygon,4326)),
  -- Налайх дүүрэг
  ('ЗТ-2022-011', ST_MakeEnvelope(107.300, 47.720, 107.360, 47.760, 4326)::geometry(Polygon,4326)),
  ('ЗТ-2024-055', ST_MakeEnvelope(107.320, 47.730, 107.370, 47.770, 4326)::geometry(Polygon,4326)),
  -- Багануур дүүрэг
  ('ЗТ-2022-019', ST_MakeEnvelope(108.220, 47.690, 108.280, 47.730, 4326)::geometry(Polygon,4326))
) AS v(plan_code, geom)
WHERE la.plan_code = v.plan_code
  AND la.geometry IS NULL;

-- ── 3. Парцелийн acquisition_geom ─────────────────────────────────────────
-- Mock өгөгдөлд парцел бүтнээрээ чөлөөлөлтийн хилийн дотор байна гэж үзэнэ.

UPDATE parcel
SET acquisition_geom = geometry
WHERE parcel_id ~ '^(11|12|13|14)-'
  AND geometry         IS NOT NULL
  AND acquisition_geom IS NULL;

-- ── 4. Хөрөнгийн geometry ─────────────────────────────────────────────────
-- Харгалзах парцелийн centroid-оос asset_number дугаараар тодорхойлсон
-- жижиг квадрат (~20×20 м ≈ 0.00018°) үүсгэнэ.

UPDATE assets a
SET geometry = ST_Envelope(
  ST_Buffer(
    ST_Translate(
      ST_Centroid(p.geometry),
      ((regexp_replace(a.asset_number, '[^0-9]', '', 'g')::bigint % 7) - 3) * 0.00006,
      ((regexp_replace(a.asset_number, '[^0-9]', '', 'g')::bigint % 5) - 2) * 0.00006
    ),
    0.00009
  )
)::geometry(Polygon, 4326)
FROM parcel p
WHERE a.parcel_id = p.parcel_id
  AND p.geometry  IS NOT NULL
  AND a.geometry  IS NULL;
