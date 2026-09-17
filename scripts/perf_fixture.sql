-- Гүйцэтгэлийн суурь хэмжилтийн FIXTURE (docs/performance-optimization-plan.md §2)
--
-- Хэмжээ нь төлөвлөгөөнд бичигдсэн суурьтай ИЖИЛ:
--   100 чөлөөлөлт · 10,000 нэгж талбар · 30,000 олговор
--   10,000 байгууламж · 50,000 статусын түүх · 10,000 үнэлгээний илгээлт
--
-- ЗӨВХӨН тусгаарласан хэмжилтийн санд (perfdb) ажиллуулна. Үндсэн appdb-д
-- ХЭЗЭЭ Ч бүү ажиллуул — эхний алхам нь бүх мөрийг устгадаг.
--
-- Ажиллуулах:
--   psql -U postgres -d perfdb -f scripts/perf_fixture.sql

\set ON_ERROR_STOP on
\timing on

BEGIN;

-- Давтагдах үр дүн: санамсаргүй утгууд ажиллуулалт болгонд ижил гарна.
SELECT setseed(0.42);

TRUNCATE parcel_valuation_submission, parcel_status_history, compensation,
         assets, land_acquisition_au, parcel, land_acquisition RESTART IDENTITY CASCADE;

-- ── Лавлахууд ───────────────────────────────────────────────────────────
INSERT INTO parcel_status (id, code, name, sort_order) VALUES
  (0,'pending','Хүлээгдэж буй',0), (1,'processing','Зөвшилцөх шатандаа',1),
  (2,'evaluating','Үнэлгээ хийх',2), (3,'affected_out','Нөлөөлөгдсөн гарсан',3),
  (4,'rejected','Татгалзсан',4), (5,'released','Чөлөөлсөн',5)
ON CONFLICT (id) DO NOTHING;

INSERT INTO acquisition_category (id, name, parent_id, sort_order) VALUES
  (1,'Гэр хороолол', NULL, 1), (2,'Авто зам', NULL, 2), (3,'Нийгмийн дэд бүтэц', NULL, 3),
  (11,'Орон сууцжуулалт', 1, 1), (12,'Замын өргөтгөл', 2, 1), (13,'Сургууль', 3, 1)
ON CONFLICT (id) DO NOTHING;

-- ── 100 чөлөөлөлт ───────────────────────────────────────────────────────
INSERT INTO land_acquisition (
  id, plan_code, acquisition_name, status, geometry, area_m2,
  general_category_id, sub_category_id, decree_number,
  professional_org_id,
  start_date, created_at, updated_at)
SELECT
  ('00000000-0000-4000-8000-' || lpad(i::text, 12, '0'))::uuid,
  'PLAN-' || lpad(i::text, 4, '0'),
  'Хэмжилтийн чөлөөлөлт ' || i,
  1 + (i % 3),                                  -- 1..3 төлөв
  ST_MakeEnvelope(
    106.80 + (i % 10) * 0.010, 47.85 + (i / 10) * 0.010,
    106.80 + (i % 10) * 0.010 + 0.0095, 47.85 + (i / 10) * 0.010 + 0.0095, 4326),
  400000 + i * 137,
  1 + (i % 3), 11 + (i % 3),
  CASE WHEN i % 4 = 0 THEN 'A-' || i ELSE '' END,
  -- Үндсэн гүйцэтгэгч байгууллага. Зарим чөлөөлөлт байгууллагагүй (NULL)
  -- байх нь мэргэжлийн байгууллагын шүүлтийн салааг бодитоор шалгана.
  CASE WHEN i % 4 = 0 THEN NULL ELSE 1 + (i % 3) END,
  DATE '2024-01-01' + ((i * 7) % 900),
  now() - ((i || ' days')::interval),
  now()
FROM generate_series(1, 100) AS i;

-- Захиргааны нэгжийн холбоос (AU2 шүүлтүүр, дашбоардын au кодод хэрэглэгдэнэ)
INSERT INTO land_acquisition_au (land_acquisition_id, au1_code, au2_code, au3_code)
SELECT la.id, '11', '11' || lpad((1 + (row_number() OVER (ORDER BY la.plan_code) % 9))::text, 2, '0'),
       '11' || lpad((1 + (row_number() OVER (ORDER BY la.plan_code) % 9))::text, 2, '0') || '01'
FROM land_acquisition la;

-- ── 10,000 нэгж талбар (чөлөөлөлт тутамд 100) ───────────────────────────
INSERT INTO parcel (
  id, parcel_id, acquisition_id, status, geometry, area_m2, acquisition_area_m2,
  au1_code, au2_code, au3_code, right_type, landuse, selected_valuation_type,
  independent_org_id, compensation_paid, created_at, updated_at)
SELECT
  ('10000000-0000-4000-8000-' || lpad(((a - 1) * 100 + p)::text, 12, '0'))::uuid,
  lpad(((a - 1) * 100 + p)::text, 12, '0'),
  ('00000000-0000-4000-8000-' || lpad(a::text, 12, '0'))::uuid,
  ((a * 7 + p) % 6),                            -- 0..5 төлөв
  ST_MakeEnvelope(
    106.80 + (a % 10) * 0.010 + (p % 10) * 0.0009,
    47.85 + (a / 10) * 0.010 + (p / 10) * 0.0009,
    106.80 + (a % 10) * 0.010 + (p % 10) * 0.0009 + 0.0008,
    47.85 + (a / 10) * 0.010 + (p / 10) * 0.0009 + 0.0008, 4326),
  500 + ((a * p) % 4000),
  300 + ((a * p) % 2500),
  '11', '11' || lpad((1 + (a % 9))::text, 2, '0'), '11' || lpad((1 + (a % 9))::text, 2, '0') || '01',
  1 + ((a + p) % 3),
  (ARRAY['Орон сууц','Үйлдвэр','Худалдаа','Гэр бүлийн хэрэгцээ'])[1 + ((a + p) % 4)],
  (ARRAY['asset','independent','mika'])[1 + ((a + p) % 3)],
  -- Хөндлөнгийн үнэлгээний байгууллага: цөөн талбарт л оноогдоно. Ингэснээр
  -- "байгууллага нь чөлөөлөлтийн үндсэн гүйцэтгэгч БИШ ч нэг талбараар
  -- холбогдсон" салаа (parcelScopeCond дахь OR) бодитоор шалгагдана.
  CASE WHEN (a + p) % 7 = 0 THEN 1 + ((a + p) % 3) ELSE NULL END,
  ((a * p) % 5 = 0),
  now() - (((a * 100 + p) % 900 || ' days')::interval),
  now()
FROM generate_series(1, 100) AS a, generate_series(1, 100) AS p;

-- ── 30,000 олговор (талбар тутамд 3) ────────────────────────────────────
INSERT INTO compensation (
  acquisition_id, parcel_id, target_type, compensation_type, valuation_type,
  coverage_percent, amount, status, note, created_by, created_at, updated_at)
SELECT
  p.acquisition_id, p.parcel_id, 'parcel',
  (ARRAY['cash','cash','land_grant'])[k],
  COALESCE(p.selected_valuation_type, 'asset'),
  100,
  1000000 + ((('x' || substr(md5(p.parcel_id || k::text), 1, 6))::bit(24)::int) % 90000000),
  (ARRAY['approved','approved','pending'])[k],
  '', 'perf-fixture', now(), now()
FROM parcel p, generate_series(1, 3) AS k;

-- ── 10,000 байгууламж (талбар тутамд 1) ─────────────────────────────────
INSERT INTO assets (
  acquisition_id, parcel_id, valuation_type, asset_number, asset_type, asset_name,
  floor_count, area_m2, owner_name, address, notes, unit, capacity, description,
  unit_price, photo_pdf_url, photo_pdf_name, created_at, updated_at)
SELECT
  p.acquisition_id, p.parcel_id, COALESCE(p.selected_valuation_type, 'asset'),
  'A-' || p.parcel_id, 'real_state', 'Барилга ' || p.parcel_id,
  1 + (length(p.parcel_id) % 5), 120 + (length(p.parcel_id) % 300),
  'Эзэмшигч ' || p.parcel_id, 'Хаяг ' || p.parcel_id, '', 'м2', '', '',
  1500000, '/api/files/asset-photo/x/p.pdf', 'p.pdf', now(), now()
FROM parcel p;

-- ── 50,000 статусын түүх (талбар тутамд 5) ──────────────────────────────
INSERT INTO parcel_status_history (
  parcel_id, acquisition_id, status_id, status_date, created_by, reason,
  attachment_url, attachment_name)
SELECT
  p.parcel_id, p.acquisition_id, (k - 1),
  now() - (((k * 60) + (length(p.parcel_id) % 400) || ' days')::interval),
  'perf-fixture', '', '', ''
FROM parcel p, generate_series(1, 5) AS k;

-- ── 10,000 үнэлгээний илгээлт (талбар тутамд 1) ─────────────────────────
INSERT INTO parcel_valuation_submission (
  acquisition_id, parcel_id, valuation_type, status,
  submitted_by, submitted_at, reviewed_by, last_note,
  attachment_url, attachment_name, created_at, updated_at)
SELECT
  p.acquisition_id, p.parcel_id, COALESCE(p.selected_valuation_type, 'asset'),
  (ARRAY['draft','submitted','approved','returned'])[1 + (length(p.parcel_id) % 4)],
  'perf-fixture', now() - interval '10 days', '', '', '', '', now(), now()
FROM parcel p;

COMMIT;

ANALYZE;

-- ── Хяналтын тоо ────────────────────────────────────────────────────────
SELECT 'land_acquisition' AS t, count(*) FROM land_acquisition
UNION ALL SELECT 'parcel', count(*) FROM parcel
UNION ALL SELECT 'compensation', count(*) FROM compensation
UNION ALL SELECT 'assets', count(*) FROM assets
UNION ALL SELECT 'parcel_status_history', count(*) FROM parcel_status_history
UNION ALL SELECT 'parcel_valuation_submission', count(*) FROM parcel_valuation_submission
ORDER BY 1;
