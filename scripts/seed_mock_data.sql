-- ============================================================
-- Mock seed: 20 чөлөөлөлт · 500 эзэмшигч · 2000 нэгж талбар
-- Ажиллуулах:  psql "$DATABASE_URL" -f scripts/seed_mock_data.sql
-- ============================================================

DO $$
DECLARE
  acq_ids    UUID[];
  holder_ids UUID[];
  ct_ids     INT[];

  acq_names TEXT[] := ARRAY[
    'Хан-Уул дүүргийн 15-р хорооны замын тэлэлт',
    'Баянзүрх дүүргийн 7-р хорооны дэд бүтцийн хөгжил',
    'Сонгинохайрхан дүүргийн 4-р хорооны орон сууцны хороолол',
    'Чингэлтэй дүүргийн 3-р хорооны гудамж тохижуулалт',
    'Сүхбаатар дүүргийн 10-р хорооны замын засвар',
    'Налайх дүүргийн 2-р хорооны ус хангамжийн систем',
    'Багануур дүүргийн 1-р хорооны цахилгаан дамжуулах шугам',
    'Баянгол дүүргийн 9-р хорооны гэрэлтүүлэг',
    'Хан-Уул дүүргийн 3-р хорооны ногоон бүс',
    'Баянзүрх дүүргийн 14-р хорооны дулаан хангамж',
    'Сонгинохайрхан дүүргийн 20-р хорооны замын сүлжээ',
    'Чингэлтэй дүүргийн 5-р хорооны цэцэрлэгийн барилга',
    'Сүхбаатар дүүргийн 4-р хорооны сургуулийн барилга',
    'Баянгол дүүргийн 5-р хорооны эрүүл мэндийн байгууллага',
    'Хан-Уул дүүргийн 8-р хорооны спорт заал',
    'Налайх дүүргийн 3-р хорооны дэд бүтцийн хөгжил',
    'Баянзүрх дүүргийн 22-р хорооны замын өргөтгөл',
    'Сонгинохайрхан дүүргийн 10-р хорооны нийтийн талбай',
    'Чингэлтэй дүүргийн 1-р хорооны орон сууцны барилга',
    'Сүхбаатар дүүргийн 1-р хорооны засаг захиргааны барилга'
  ];

  plan_codes TEXT[] := ARRAY[
    'ЗТ-2023-001','ЗТ-2023-015','ЗТ-2024-003','ЗТ-2024-022','ЗТ-2024-007',
    'ЗТ-2022-011','ЗТ-2022-019','ЗТ-2023-008','ЗТ-2024-031','ЗТ-2023-042',
    'ЗТ-2021-005','ЗТ-2025-002','ЗТ-2024-018','ЗТ-2023-027','ЗТ-2022-034',
    'ЗТ-2024-055','ЗТ-2023-071','ЗТ-2022-088','ЗТ-2024-099','ЗТ-2023-112'
  ];

  last_names TEXT[] := ARRAY[
    'Батбаяр','Дорж','Мөнх','Ганбаатар','Цэрэн',
    'Нямдорж','Жаргал','Пүрэв','Лхагва','Балган',
    'Гомбо','Сандаг','Цэдэн','Дамба','Сүрэн',
    'Бат','Ган','Энх','Мөнхбаяр','Гантөмөр'
  ];

  first_names TEXT[] := ARRAY[
    'Батболд','Энхбаяр','Мөнхбаяр','Ганболд','Цэрэнпунцаг',
    'Нямсүрэн','Жаргалмаа','Пүрэвдорж','Лхагвасүрэн','Балганцэрэн',
    'Гомбосүрэн','Сандагдорж','Цэдэндамба','Дамбажав','Сүрэнхорлоо',
    'Оюунбаатар','Нарантуяа','Мөнхзул','Болормаа','Энхтуяа'
  ];

  reg_pfx TEXT[] := ARRAY[
    'АА','АБ','АВ','АГ','АД','БА','ББ','БВ',
    'ВА','ВБ','ГА','ДА','ЕА','ЖА','ЗА'
  ];

  -- 4 дүүрэг: 11=Хан-Уул 12=Баянзүрх 13=Сонгинохайрхан 14=Чингэлтэй
  dist_ch TEXT[] := ARRAY['11','12','13','14'];

  i        INT;
  dg       INT;   -- district group (0-3)
  li       INT;   -- local index within district group (0-499)
  khoroo   INT;   -- 1-10
  blk      INT;   -- 1-10
  pnum     INT;   -- 1-5
  dcode    TEXT;
  p_id     TEXT;  -- parcel_id string, 12 chars
  prow_id  UUID;  -- parcel.id (UUID)
  acq_id   UUID;
  hld_id   UUID;
  area_tot NUMERIC;
  acq_area NUMERIC;
  rem_area NUMERIC;
  lcomp    NUMERIC;
  acomp    NUMERIC;
  pcomp    NUMERIC;
  ast_id   UUID;
BEGIN

  -- ── 1. Construction type IDs ─────────────────────────────────────────────
  SELECT ARRAY(SELECT id FROM construction_type ORDER BY sort_order) INTO ct_ids;
  IF ct_ids IS NULL OR array_length(ct_ids, 1) = 0 THEN
    RAISE EXCEPTION 'construction_type хоосон байна — migration 000017 эхлээд ажиллуулна уу';
  END IF;

  -- ── 2. Land acquisition: 20 мөр ──────────────────────────────────────────
  WITH ins AS (
    INSERT INTO land_acquisition (
      plan_code, plan_name, acquisition_name, construction_type_id,
      decree_number, decree_date, start_date, end_date, status,
      implementing_org, responsible_org, funding_source
    )
    SELECT
      plan_codes[s],
      acq_names[s],
      acq_names[s],
      ct_ids[((s - 1) % array_length(ct_ids, 1)) + 1],
      'А/' || lpad((s * 47)::TEXT, 3, '0'),
      make_date(2020 + ((s - 1) % 5), ((s * 3 - 1) % 12) + 1, ((s * 7 - 1) % 28) + 1),
      make_date(2020 + ((s - 1) % 5), 1, 1),
      make_date(2025 + ((s - 1) % 3), 12, 31),
      3,
      CASE s % 4
        WHEN 0 THEN 'Нийслэлийн Зам, тээврийн газар'
        WHEN 1 THEN 'Нийслэлийн Барилга, хот байгуулалтын газар'
        WHEN 2 THEN 'Нийслэлийн Өмч, газрын харилцааны газар'
        ELSE 'Улаанбаатар хотын Засаг даргын тамгын газар'
      END,
      CASE s % 3
        WHEN 0 THEN 'Нийслэлийн Засаг дарга'
        WHEN 1 THEN 'Хан-Уул дүүргийн Засаг дарга'
        ELSE 'Баянзүрх дүүргийн Засаг дарга'
      END,
      CASE s % 3
        WHEN 0 THEN 'Улсын төсөв'
        WHEN 1 THEN 'Нийслэлийн төсөв'
        ELSE 'Гадаадын зээл, тусламж'
      END
    FROM generate_series(1, 20) AS s
    RETURNING id
  )
  SELECT array_agg(id) INTO acq_ids FROM ins;

  RAISE NOTICE '20 чөлөөлөлт үүслээ';

  -- ── 3. Right holder: 500 мөр ─────────────────────────────────────────────
  WITH ins AS (
    INSERT INTO right_holder (last_name, name, register_no, holder_type)
    SELECT
      last_names[((s - 1) % 20) + 1],
      first_names[((s * 7 - 1) % 20) + 1],
      reg_pfx[((s * 3 - 1) % 15) + 1]
        || lpad(((50 + ((s * 13) % 55))::TEXT), 2, '0')
        || lpad((((s * 5) % 12) + 1)::TEXT, 2, '0')
        || lpad((((s * 7) % 28) + 1)::TEXT, 2, '0')
        || lpad(((s % 99) + 1)::TEXT, 2, '0'),
      CASE s % 3 WHEN 0 THEN 'company' ELSE 'person' END
    FROM generate_series(1, 500) AS s
    RETURNING id
  )
  SELECT array_agg(id) INTO holder_ids FROM ins;

  RAISE NOTICE '500 эзэмшигч үүслээ';

  -- ── 4. Parcel + parcel_detail + compensation: 2000 мөр ───────────────────
  --
  -- Parcel ID формат: "DD-KK-BB-PPP" (яг 12 тэмдэгт)
  --   DD  = дүүрэг (11-14)
  --   KK  = хороо (01-10)
  --   BB  = блок  (01-10)
  --   PPP = парцелийн дугаар (001-005)
  --
  -- Хуваарилалт:
  --   i 1-500   → дүүрэг 11, 10 хороо × 10 блок × 5 парцел = 500
  --   i 501-1000 → дүүрэг 12, мөн адил
  --   i 1001-1500 → дүүрэг 13
  --   i 1501-2000 → дүүрэг 14
  --
  FOR i IN 1..2000 LOOP
    dg     := (i - 1) / 500;           -- 0,1,2,3
    li     := (i - 1) % 500;           -- 0-499
    khoroo := li / 50 + 1;             -- 1-10
    blk    := (li % 50) / 5 + 1;       -- 1-10
    pnum   := li % 5 + 1;              -- 1-5
    dcode  := dist_ch[dg + 1];

    p_id := dcode
      || '-' || lpad(khoroo::TEXT, 2, '0')
      || '-' || lpad(blk::TEXT,    2, '0')
      || '-' || lpad(pnum::TEXT,   3, '0');  -- 12 тэмдэгт

    acq_id   := acq_ids[((i - 1) % 20) + 1];
    area_tot := 100 + ((i * 43) % 4900);
    acq_area := 10  + ((i * 23) % (area_tot::INT - 10));
    rem_area := area_tot - acq_area;

    -- 4a. Нэгж талбар
    INSERT INTO parcel (
      parcel_id, acquisition_id,
      au1_code, au2_code, au3_code,
      area_m2, acquisition_area_m2, remaining_area_m2,
      right_type, compensation_paid, db_changed, changed_parcel_id,
      valid_from, valid_till
    ) VALUES (
      p_id, acq_id,
      '01',
      '01' || dcode,
      dcode || lpad(khoroo::TEXT, 2, '0'),
      area_tot, acq_area, rem_area,
      (i % 3) + 1,
      (i % 7 = 0),
      (i % 5 = 0),
      CASE WHEN i % 5 = 0
        THEN dcode
          || '-' || lpad(khoroo::TEXT, 2, '0')
          || '-' || lpad((blk % 10 + 1)::TEXT, 2, '0')
          || '-' || lpad(((pnum % 5) + 1)::TEXT, 3, '0')
        ELSE ''
      END,
      make_date(2015 + ((i * 3) % 10), ((i * 5) % 12) + 1, ((i * 7) % 28) + 1),
      make_date(2045, 12, 31)
    ) RETURNING id INTO prow_id;

    -- 4b. Эрхийн дэлгэрэнгүй (holder холбоос)
    hld_id := holder_ids[((i - 1) % 500) + 1];
    INSERT INTO parcel_detail (parcel_id, acquisition_id, right_type, holder_id)
    VALUES (prow_id, acq_id, (i % 3) + 1, hld_id);

    -- 4c. Газрын нөхөн төлбөр (target_type='parcel' — бүх 2000-д байна)
    lcomp := 1000000 + ((i * 53) % 499000000)::NUMERIC;
    INSERT INTO compensation (
      acquisition_id, parcel_id, target_type, compensation_type, amount, coverage_percent
    ) VALUES (
      acq_id, p_id, 'parcel',
      CASE i % 3 WHEN 2 THEN 'land_grant' ELSE 'cash' END,
      lcomp, 100
    );

    -- 4d. Үл хөдлөх хөрөнгийн нөхөн төлбөр (~1333 парцел, i % 3 != 0)
    IF i % 3 != 0 THEN
      acomp := 500000 + ((i * 61) % 199500000)::NUMERIC;
      INSERT INTO assets (
        acquisition_id, parcel_id, asset_number, asset_type, asset_name, floor_count, area_m2
      ) VALUES (
        acq_id, p_id,
        'БД-' || lpad(i::TEXT, 5, '0'),
        'real_state',
        CASE i % 4
          WHEN 0 THEN 'Орон сууцны байшин'
          WHEN 1 THEN 'Үйлчилгээний барилга'
          WHEN 2 THEN 'Цех, агуулах'
          ELSE 'Бусад барилга байгууламж'
        END,
        (i % 5) + 1,
        20 + ((i * 37) % 480)
      ) RETURNING id INTO ast_id;

      INSERT INTO compensation (
        acquisition_id, parcel_id, target_type, asset_id, compensation_type, amount, coverage_percent
      ) VALUES (
        acq_id, p_id, 'asset', ast_id, 'cash', acomp,
        CASE i % 3 WHEN 1 THEN 100 ELSE 80 END
      );
    END IF;

    -- 4e. Эд хөрөнгийн нөхөн төлбөр (~500 парцел, i % 4 = 0)
    IF i % 4 = 0 THEN
      pcomp := 100000 + ((i * 71) % 49900000)::NUMERIC;
      INSERT INTO assets (
        acquisition_id, parcel_id, asset_number, asset_type, asset_name, area_m2
      ) VALUES (
        acq_id, p_id,
        'ЭД-' || lpad(i::TEXT, 5, '0'),
        'property',
        CASE i % 3
          WHEN 0 THEN 'Хашаа, хороо'
          WHEN 1 THEN 'Худаг, шатан'
          ELSE 'Тоног төхөөрөмж'
        END,
        10 + ((i * 13) % 90)
      ) RETURNING id INTO ast_id;

      INSERT INTO compensation (
        acquisition_id, parcel_id, target_type, asset_id, compensation_type, amount, coverage_percent
      ) VALUES (
        acq_id, p_id, 'asset', ast_id, 'cash', pcomp, 100
      );
    END IF;

  END LOOP;

  RAISE NOTICE 'Seed дууслаа: 20 чөлөөлөлт, 500 эзэмшигч, 2000 нэгж талбар';
END $$;
