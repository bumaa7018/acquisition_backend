-- ============================================================
-- Seed: нөхөх төлбөрийн жишээ өгөгдөл
-- Ажиллуулах: psql "$DATABASE_URL" -f scripts/seed_compensation_samples.sql
-- Idempotent: [seed:compensation] marker-ээр давхардуулахгүй.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
DECLARE
  marker       TEXT := '[seed:compensation]';
  acq_id       UUID;
  parcel_code  VARCHAR(12);
  real_asset   UUID;
  prop_asset   UUID;
  comp_id      UUID;
BEGIN
  SELECT p.acquisition_id, p.parcel_id
  INTO acq_id, parcel_code
  FROM parcel p
  ORDER BY p.created_at, p.parcel_id
  LIMIT 1;

  IF acq_id IS NULL OR parcel_code IS NULL THEN
    RAISE EXCEPTION 'Нөхөх төлбөр seed хийх нэгж талбар олдсонгүй';
  END IF;

  INSERT INTO assets (
    acquisition_id, parcel_id, asset_number, asset_type, asset_name,
    floor_count, area_m2, owner_name, address, notes
  )
  SELECT
    acq_id, parcel_code, 'SEED-BLD-001', 'real_state', 'Амины орон сууц',
    1, 64, 'Дамдинсүрэн Болд', 'СХД, 5-р хороо, жишээ хаяг 12',
    marker || ' Үл хөдлөх хөрөнгийн жишээ өгөгдөл'
  WHERE NOT EXISTS (
    SELECT 1 FROM assets
    WHERE acquisition_id = acq_id
      AND parcel_id = parcel_code
      AND asset_number = 'SEED-BLD-001'
  );

  INSERT INTO assets (
    acquisition_id, parcel_id, asset_number, asset_type, asset_name,
    floor_count, area_m2, owner_name, address, notes
  )
  SELECT
    acq_id, parcel_code, 'SEED-PRP-001', 'property', 'Хашаа, худгийн байгууламж',
    0, 28, 'Дамдинсүрэн Болд', 'СХД, 5-р хороо, жишээ хаяг 12',
    marker || ' Эд хөрөнгийн жишээ өгөгдөл'
  WHERE NOT EXISTS (
    SELECT 1 FROM assets
    WHERE acquisition_id = acq_id
      AND parcel_id = parcel_code
      AND asset_number = 'SEED-PRP-001'
  );

  SELECT id INTO real_asset
  FROM assets
  WHERE acquisition_id = acq_id
    AND parcel_id = parcel_code
    AND asset_number = 'SEED-BLD-001'
  LIMIT 1;

  SELECT id INTO prop_asset
  FROM assets
  WHERE acquisition_id = acq_id
    AND parcel_id = parcel_code
    AND asset_number = 'SEED-PRP-001'
  LIMIT 1;

  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    acq_id, parcel_code, 'parcel', 'cash',
    100, 31500000, DATE '2026-06-15',
    marker || ' Нэгж талбарын мөнгөн нөхөн төлбөр',
    'seed'
  WHERE NOT EXISTS (
    SELECT 1 FROM compensation
    WHERE acquisition_id = acq_id
      AND parcel_id = parcel_code
      AND note = marker || ' Нэгж талбарын мөнгөн нөхөн төлбөр'
  );

  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, asset_id, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    acq_id, parcel_code, 'asset', real_asset, 'cash',
    100, 45200000, DATE '2026-06-16',
    marker || ' Хөрөнгийн мөнгөн нөхөн төлбөр',
    'seed'
  WHERE real_asset IS NOT NULL
    AND NOT EXISTS (
      SELECT 1 FROM compensation
      WHERE acquisition_id = acq_id
        AND parcel_id = parcel_code
        AND note = marker || ' Хөрөнгийн мөнгөн нөхөн төлбөр'
    );

  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, asset_id, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    acq_id, parcel_code, 'asset', prop_asset, 'cash',
    100, 8500000, DATE '2026-06-16',
    marker || ' Эд хөрөнгийн мөнгөн нөхөн төлбөр',
    'seed'
  WHERE prop_asset IS NOT NULL
    AND NOT EXISTS (
      SELECT 1 FROM compensation
      WHERE acquisition_id = acq_id
        AND parcel_id = parcel_code
        AND note = marker || ' Эд хөрөнгийн мөнгөн нөхөн төлбөр'
    );

  INSERT INTO compensation (
    acquisition_id, parcel_id, target_type, compensation_type,
    coverage_percent, amount, compensation_date, note, created_by
  )
  SELECT
    acq_id, parcel_code, 'parcel', 'land_grant',
    70, 24000000, DATE '2026-06-17',
    marker || ' Газраар дүйцүүлэх нөхөх олговор',
    'seed'
  WHERE NOT EXISTS (
    SELECT 1 FROM compensation
    WHERE acquisition_id = acq_id
      AND parcel_id = parcel_code
      AND note = marker || ' Газраар дүйцүүлэх нөхөх олговор'
  )
  RETURNING id INTO comp_id;

  IF comp_id IS NULL THEN
    SELECT id INTO comp_id
    FROM compensation
    WHERE acquisition_id = acq_id
      AND parcel_id = parcel_code
      AND note = marker || ' Газраар дүйцүүлэх нөхөх олговор'
    LIMIT 1;
  END IF;

  IF comp_id IS NOT NULL THEN
    INSERT INTO compensation_grant (
      acquisition_id, compensation_id, amount, grant_date, note,
      land_area_m2, land_price, land_location, land_purpose,
      land_use_type, parcel_number, decree_number, created_by
    )
    SELECT
      acq_id, comp_id, 24000000, DATE '2026-06-20',
      marker || ' Газрын нөхөх олговрын дэлгэрэнгүй',
      420, 57143, 'СХД, 21-р хороо', 'Гэр бүлийн хэрэгцээ',
      'Эзэмших', 'SEED-GRANT-001', 'А/2026-001', 'seed'
    WHERE NOT EXISTS (
      SELECT 1 FROM compensation_grant
      WHERE compensation_id = comp_id
        AND note = marker || ' Газрын нөхөх олговрын дэлгэрэнгүй'
    );
  END IF;

  UPDATE parcel
  SET compensation_paid = true
  WHERE parcel_id = parcel_code;

  RAISE NOTICE 'Нөхөх төлбөрийн seed дууслаа: acquisition %, parcel %', acq_id, parcel_code;
END $$;
