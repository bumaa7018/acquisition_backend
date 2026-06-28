-- ============================================================
-- Seed: бүх нэгж талбарын үнэлгээний жишээ өгөгдөл
-- Ажиллуулах: psql "$DATABASE_URL" -f scripts/seed_valuation_samples.sql
-- Idempotent: [seed:valuation] marker-ээр давхардуулахгүй.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
DECLARE
  marker               TEXT := '[seed:valuation]';
  parcel_total         INT;
  detail_inserted      INT := 0;
  valuation_updated    INT := 0;
  asset_inserted       INT := 0;
  compensation_inserted INT := 0;
  grant_inserted       INT := 0;
  row_count            INT := 0;
BEGIN
  SELECT COUNT(*) INTO parcel_total FROM parcel;

  IF parcel_total = 0 THEN
    RAISE EXCEPTION 'Үнэлгээ seed хийх нэгж талбар олдсонгүй';
  END IF;

  INSERT INTO parcel_detail (
    parcel_id, acquisition_id, right_type,
    app_no, decision_no, contract_no, certificate_no, created_by
  )
  SELECT
    p.id,
    p.acquisition_id,
    COALESCE(p.right_type, 1),
    'APP-' || p.parcel_id,
    'DEC-' || p.parcel_id,
    'CON-' || p.parcel_id,
    'CERT-' || p.parcel_id,
    'seed'
  FROM parcel p
  WHERE NOT EXISTS (
    SELECT 1
    FROM parcel_detail pd
    WHERE pd.parcel_id = p.id
  );
  GET DIAGNOSTICS detail_inserted = ROW_COUNT;

  WITH ranked AS (
    SELECT
      p.id,
      p.parcel_id,
      p.acquisition_area_m2,
      ROW_NUMBER() OVER (ORDER BY p.created_at, p.parcel_id) AS rn
    FROM parcel p
  )
  UPDATE parcel_detail pd
  SET
    valuation_zone = COALESCE(NULLIF(pd.valuation_zone, ''), 'Үнэлгээний бүс ' || (((ranked.rn - 1) % 5) + 1)),
    base_price_per_ha = COALESCE(pd.base_price_per_ha, 18000000 + (((ranked.rn * 137) % 12000000))::NUMERIC),
    auction_coeff = COALESCE(pd.auction_coeff, 1 + (((ranked.rn % 7)::NUMERIC) / 10)),
    auction_price = COALESCE(
      pd.auction_price,
      ROUND((18000000 + (((ranked.rn * 137) % 12000000))::NUMERIC) * (1 + (((ranked.rn % 7)::NUMERIC) / 10)), 4)
    ),
    updated_at = NOW(),
    updated_by = 'seed'
  FROM ranked
  WHERE pd.parcel_id = ranked.id;
  GET DIAGNOSTICS valuation_updated = ROW_COUNT;

  WITH ranked AS (
    SELECT
      p.acquisition_id,
      p.parcel_id,
      p.area_m2,
      p.acquisition_area_m2,
      ROW_NUMBER() OVER (ORDER BY p.created_at, p.parcel_id) AS rn
    FROM parcel p
  )
  INSERT INTO assets (
    acquisition_id, parcel_id, asset_number, asset_type, asset_name,
    floor_count, area_m2, owner_name, address, notes
  )
  SELECT
    r.acquisition_id,
    r.parcel_id,
    'VAL-BLD-' || r.parcel_id,
    'real_state',
    CASE r.rn % 4
      WHEN 0 THEN 'Амины орон сууц'
      WHEN 1 THEN 'Үйлчилгээний барилга'
      WHEN 2 THEN 'Агуулах'
      ELSE 'Гараж'
    END,
    (r.rn % 3) + 1,
    40 + ((r.rn * 17) % 160),
    'Seed эзэмшигч ' || r.rn,
    'Жишээ хаяг, нэгж талбар ' || r.parcel_id,
    marker || ' Үл хөдлөх хөрөнгийн үнэлгээний жишээ'
  FROM ranked r
  WHERE NOT EXISTS (
    SELECT 1
    FROM assets a
    WHERE a.acquisition_id = r.acquisition_id
      AND a.parcel_id = r.parcel_id
      AND a.asset_number = 'VAL-BLD-' || r.parcel_id
  );
  GET DIAGNOSTICS asset_inserted = ROW_COUNT;

  WITH ranked AS (
    SELECT
      p.acquisition_id,
      p.parcel_id,
      ROW_NUMBER() OVER (ORDER BY p.created_at, p.parcel_id) AS rn
    FROM parcel p
  )
  INSERT INTO assets (
    acquisition_id, parcel_id, asset_number, asset_type, asset_name,
    floor_count, area_m2, owner_name, address, notes
  )
  SELECT
    r.acquisition_id,
    r.parcel_id,
    'VAL-PRP-' || r.parcel_id,
    'property',
    CASE r.rn % 3
      WHEN 0 THEN 'Хашаа'
      WHEN 1 THEN 'Худаг'
      ELSE 'Тоног төхөөрөмж'
    END,
    0,
    10 + ((r.rn * 11) % 70),
    'Seed эзэмшигч ' || r.rn,
    'Жишээ хаяг, нэгж талбар ' || r.parcel_id,
    marker || ' Эд хөрөнгийн үнэлгээний жишээ'
  FROM ranked r
  WHERE NOT EXISTS (
    SELECT 1
    FROM assets a
    WHERE a.acquisition_id = r.acquisition_id
      AND a.parcel_id = r.parcel_id
      AND a.asset_number = 'VAL-PRP-' || r.parcel_id
  );
  GET DIAGNOSTICS row_count = ROW_COUNT;
  asset_inserted := asset_inserted + row_count;

  WITH ranked AS (
    SELECT
      p.acquisition_id,
      p.parcel_id,
      p.acquisition_area_m2,
      ROW_NUMBER() OVER (ORDER BY p.created_at, p.parcel_id) AS rn
    FROM parcel p
  )
  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    r.acquisition_id,
    r.parcel_id,
    'parcel',
    'cash',
    100,
    ROUND((COALESCE(r.acquisition_area_m2, 0) * (55000 + ((r.rn * 97) % 45000)))::NUMERIC, 2),
    DATE '2026-06-15',
    marker || ' Нэгж талбарын мөнгөн үнэлгээ',
    'seed'
  FROM ranked r
  WHERE NOT EXISTS (
    SELECT 1
    FROM compensation c
    WHERE c.acquisition_id = r.acquisition_id
      AND c.parcel_id = r.parcel_id
      AND c.note = marker || ' Нэгж талбарын мөнгөн үнэлгээ'
  );
  GET DIAGNOSTICS compensation_inserted = ROW_COUNT;

  WITH asset_rows AS (
    SELECT
      a.id,
      a.acquisition_id,
      a.parcel_id,
      a.asset_type,
      ROW_NUMBER() OVER (ORDER BY a.created_at, a.parcel_id, a.asset_number) AS rn
    FROM assets a
    WHERE a.notes LIKE marker || '%'
  )
  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, asset_id, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    a.acquisition_id,
    a.parcel_id,
    'asset',
    a.id,
    'cash',
    CASE WHEN a.asset_type = 'real_state' THEN 100 ELSE 90 END,
    CASE
      WHEN a.asset_type = 'real_state' THEN 18000000 + ((a.rn * 421) % 42000000)
      ELSE 2500000 + ((a.rn * 211) % 8500000)
    END,
    DATE '2026-06-16',
    CASE
      WHEN a.asset_type = 'real_state' THEN marker || ' Хөрөнгийн мөнгөн үнэлгээ'
      ELSE marker || ' Эд хөрөнгийн мөнгөн үнэлгээ'
    END,
    'seed'
  FROM asset_rows a
  WHERE NOT EXISTS (
    SELECT 1
    FROM compensation c
    WHERE c.acquisition_id = a.acquisition_id
      AND c.parcel_id = a.parcel_id
      AND c.asset_id = a.id
      AND c.note IN (
        marker || ' Хөрөнгийн мөнгөн үнэлгээ',
        marker || ' Эд хөрөнгийн мөнгөн үнэлгээ'
      )
  );
  GET DIAGNOSTICS row_count = ROW_COUNT;
  compensation_inserted := compensation_inserted + row_count;

  WITH ranked AS (
    SELECT
      p.acquisition_id,
      p.parcel_id,
      p.acquisition_area_m2,
      ROW_NUMBER() OVER (ORDER BY p.created_at, p.parcel_id) AS rn
    FROM parcel p
  )
  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    r.acquisition_id,
    r.parcel_id,
    'parcel',
    'land_grant',
    70,
    ROUND((COALESCE(r.acquisition_area_m2, 0) * (38000 + ((r.rn * 73) % 27000)))::NUMERIC, 2),
    DATE '2026-06-17',
    marker || ' Газраар дүйцүүлэх үнэлгээ',
    'seed'
  FROM ranked r
  WHERE NOT EXISTS (
    SELECT 1
    FROM compensation c
    WHERE c.acquisition_id = r.acquisition_id
      AND c.parcel_id = r.parcel_id
      AND c.note = marker || ' Газраар дүйцүүлэх үнэлгээ'
  );
  GET DIAGNOSTICS row_count = ROW_COUNT;
  compensation_inserted := compensation_inserted + row_count;

  WITH grant_comps AS (
    SELECT
      c.id,
      c.acquisition_id,
      c.parcel_id,
      c.amount,
      ROW_NUMBER() OVER (ORDER BY c.created_at, c.parcel_id) AS rn
    FROM compensation c
    WHERE c.note = marker || ' Газраар дүйцүүлэх үнэлгээ'
  )
  INSERT INTO compensation_grant (
    acquisition_id, compensation_id, amount, grant_date, note,
    land_area_m2, land_price, land_location, land_purpose,
    land_use_type, parcel_number, decree_number, created_by
  )
  SELECT
    g.acquisition_id,
    g.id,
    g.amount,
    DATE '2026-06-20',
    marker || ' Газрын олговрын дэлгэрэнгүй',
    300 + ((g.rn * 13) % 700),
    55000 + ((g.rn * 97) % 45000),
    'Жишээ байршил ' || ((g.rn % 9) + 1),
    'Гэр бүлийн хэрэгцээ',
    'Эзэмших',
    'VAL-GRANT-' || g.parcel_id,
    'А/2026-' || LPAD(g.rn::TEXT, 4, '0'),
    'seed'
  FROM grant_comps g
  WHERE NOT EXISTS (
    SELECT 1
    FROM compensation_grant cg
    WHERE cg.compensation_id = g.id
      AND cg.note = marker || ' Газрын олговрын дэлгэрэнгүй'
  );
  GET DIAGNOSTICS grant_inserted = ROW_COUNT;

  RAISE NOTICE
    'Үнэлгээний seed дууслаа: parcels=%, parcel_detail_inserted=%, valuations_updated=%, assets_inserted=%, compensations_inserted=%, grants_inserted=%',
    parcel_total, detail_inserted, valuation_updated, asset_inserted, compensation_inserted, grant_inserted;
END $$;
