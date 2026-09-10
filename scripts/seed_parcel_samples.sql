-- ============================================================
-- Seed: нэгж талбарын ЖИШЭЭ өгөгдөл (өмчлөгч + байршлын давхардал)
--
-- Ажиллуулах:  make seed-parcel-samples
--              (эсвэл: go run ./cmd/seed-sql -db main scripts/seed_parcel_samples.sql)
--
-- Юуг харагдуулах вэ:
--   1. Чөлөөлөлтийн НЭГЖ ТАЛБАРЫН ЖАГСААЛТ дээрх "Өмчлөгч, эзэмшигч" багана.
--      Эзэмшигч нь ГУС-аас (SyncParcel) татагддаг тул шинэ санд хоосон байдаг —
--      багана бүхэлдээ "—" харагдана. Энэ seed нь эзэмшигчгүй нэгж талбарт
--      жишээ өмчлөгч нэмнэ; хэсэг талбарт 2-3 эзэмшигч оноож "+N" тэмдэг
--      харагдуулна.
--   2. БАЙРШЛЫН ДАВХАРДЛЫН улаан тодруулга ба "давхардалтай эсэх" шүүлтүүр.
--      Давхардал нь нэгж талбар ТАТАХ мөчид тооцогддог тул аль хэдийн
--      татагдсан өгөгдөл дээр parcel_overlap хоосон байдаг. Энэ seed нь
--      DetectParcelOverlaps-ийн ЯГ ижил тооцоог бүх чөлөөлөлт дээр гүйцэтгэнэ.
--
-- Idempotent: жишээ өмчлөгчийг `created_by = '[seed:parcel-samples]'`-ээр
-- танина — дахин ажиллуулахад давхардуулахгүй. Давхардлын тооцоо нь
-- үүсмэл өгөгдөл тул үргэлж дахин бодогдоно (аппын логиктой ижил).
--
-- ГУС-аас ТАТСАН өгөгдлийг ХӨНДӨХГҮЙ: эзэмшигчтэй нэгж талбар хэвээр үлдэнэ.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
DECLARE
  marker            TEXT := '[seed:parcel-samples]';
  -- Давхардлыг ХҮЛЭЭН ЗӨВШӨӨРӨХ доод хязгаар (м²).
  -- repo-гийн overlapMinAreaM2-тай ИЖИЛ байх ЁСТОЙ: хөрш талбарууд нийтлэг
  -- хилээрээ шүргэлцэхэд хэмжилтийн бөөрөнхийлөлтөөс хэдхэн см² "давхцал"
  -- гардаг; түүнийг давхардал гэж бүртгэвэл бүх хөрш талбар улаан болно.
  min_overlap_m2    NUMERIC := 1.0;
  parcel_total      INT;
  without_holder    INT;
  holders_inserted  INT := 0;
  links_inserted    INT := 0;
  details_linked    INT := 0;
  overlap_rows      INT := 0;
  overlap_parcels   INT := 0;
BEGIN
  SELECT COUNT(*) INTO parcel_total FROM parcel WHERE deleted_at IS NULL;
  IF parcel_total = 0 THEN
    RAISE NOTICE 'Нэгж талбар байхгүй — seed хийх зүйл алга';
    RETURN;
  END IF;

  -- ── 1. ЖИШЭЭ ӨМЧЛӨГЧ / ЭЗЭМШИГЧ ────────────────────────────────────────
  --
  -- Эзэмшигч БАЙХГҮЙ нэгж талбаруудыг эрэмбэлж, дугаарын үлдэгдлээр
  -- эзэмшигчийн тоог хуваарилна:
  --    rn % 7 = 0 → 3 эзэмшигч   (жагсаалтад "+2" харагдана)
  --    rn % 5 = 0 → 2 эзэмшигч   ("+1")
  --    бусад      → 1 эзэмшигч
  -- Ингэснээр "+N" тэмдэг болон нэг эзэмшигчтэй хэвийн мөр хоёулаа харагдана.
  WITH target AS (
    SELECT p.id, p.acquisition_id, p.parcel_id,
           ROW_NUMBER() OVER (ORDER BY p.acquisition_id, p.parcel_id) AS rn
      FROM parcel p
     WHERE p.deleted_at IS NULL
       AND NOT EXISTS (
             SELECT 1 FROM parcel_holder ph
              WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder')
  ),
  expanded AS (
    -- Эзэмшигчийн тоогоор мөр үржүүлнэ (1..n).
    SELECT t.*, gs.i
      FROM target t
      CROSS JOIN LATERAL generate_series(
        1,
        CASE WHEN t.rn % 7 = 0 THEN 3
             WHEN t.rn % 5 = 0 THEN 2
             ELSE 1 END
      ) AS gs(i)
  ),
  named AS (
    SELECT e.*,
           -- Овог/нэрийг дугаараас тогтмол (deterministic) сонгоно — дахин
           -- ажиллуулахад ижил нэр гарна.
           (ARRAY['Батбаяр','Дорж','Цэрэн','Сүхбаатар','Ганбат','Оюун',
                  'Энхтүвшин','Мөнхбат','Алтанцэцэг','Нэргүй','Пүрэв','Дулмаа'
            ])[1 + ((e.rn * 3 + e.i) % 12)] AS last_name,
           (ARRAY['Болормаа','Тэмүүлэн','Ануужин','Хүслэн','Мандах','Эрдэнэ',
                  'Сарнай','Batbold','Namuun','Тэгшзаяа','Дэлгэрмаа','Хонгор'
            ])[1 + ((e.rn * 5 + e.i * 2) % 12)] AS first_name
      FROM expanded e
  ),
  ins AS (
    INSERT INTO right_holder (
      last_name, name, register_no, phone, email, holder_type,
      created_by, updated_by
    )
    SELECT
      n.last_name,
      n.first_name,
      -- Регистр: 2 кирилл үсэг + 8 цифр (бодит хэлбэртэй, зохиомол утга)
      (ARRAY['ЖА','УХ','МД','ЧМ','ТГ','ХП','ЕА','ВЮ'])[1 + (n.rn % 8)]
        || LPAD(((n.rn * 7 + n.i * 13) % 99999999)::text, 8, '0'),
      '99' || LPAD(((n.rn * 11 + n.i) % 999999)::text, 6, '0'),
      '',
      'citizen',
      marker, marker
    FROM named n
    RETURNING id, last_name, name, created_by
  )
  SELECT COUNT(*) INTO holders_inserted FROM ins;

  -- Дээрх INSERT-ийн мөрүүдийг нэгж талбартай холбоно. right_holder нь
  -- нэгж талбарын мэдээлэл агуулдаггүй тул холбоосыг ДАХИН тооцож,
  -- эзэмшигчгүй талбар + шинэ (marker-тэй) эзэмшигчийг эрэмбээрээ тааруулна.
  WITH target AS (
    SELECT p.id, p.acquisition_id, p.parcel_id,
           ROW_NUMBER() OVER (ORDER BY p.acquisition_id, p.parcel_id) AS rn
      FROM parcel p
     WHERE p.deleted_at IS NULL
       AND NOT EXISTS (
             SELECT 1 FROM parcel_holder ph
              WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder')
  ),
  expanded AS (
    SELECT t.*, gs.i,
           ROW_NUMBER() OVER (ORDER BY t.rn, gs.i) AS seq
      FROM target t
      CROSS JOIN LATERAL generate_series(
        1,
        CASE WHEN t.rn % 7 = 0 THEN 3
             WHEN t.rn % 5 = 0 THEN 2
             ELSE 1 END
      ) AS gs(i)
  ),
  free_holders AS (
    -- Холбоосгүй, marker-тэй эзэмшигчид (яг одоо үүссэн мөрүүд).
    SELECT rh.id,
           ROW_NUMBER() OVER (ORDER BY rh.created_at, rh.id) AS seq
      FROM right_holder rh
     WHERE rh.created_by = marker
       AND NOT EXISTS (SELECT 1 FROM parcel_holder ph WHERE ph.holder_id = rh.id)
  ),
  link AS (
    INSERT INTO parcel_holder (
      parcel_id, acquisition_id, holder_id, app_no,
      main_applicant, holder_role, payment_recipient, note,
      created_by, updated_by
    )
    SELECT e.id, e.acquisition_id, f.id, 'APP-' || e.parcel_id,
           e.i = 1,            -- ҮНДСЭН хүсэлт гаргагч нь ЭХНИЙ эзэмшигч
           'holder', e.i = 1, '',
           marker, marker
      FROM expanded e
      JOIN free_holders f ON f.seq = e.seq
    RETURNING parcel_id, holder_id, main_applicant
  )
  SELECT COUNT(*) INTO links_inserted FROM link;

  -- parcel_detail.holder_id — тайлан (ListReportParcels) ҮҮГЭЭР эзэмшигчийн
  -- нэрийг гаргадаг тул үндсэн эзэмшигчид холбоно (хоосон байвал).
  WITH main_holder AS (
    SELECT ph.parcel_id, ph.holder_id
      FROM parcel_holder ph
     WHERE ph.holder_role = 'holder' AND ph.main_applicant
  ),
  upd AS (
    UPDATE parcel_detail pd
       SET holder_id = m.holder_id, updated_at = NOW(), updated_by = marker
      FROM main_holder m
     WHERE pd.parcel_id = m.parcel_id AND pd.holder_id IS NULL
    RETURNING pd.id
  )
  SELECT COUNT(*) INTO details_linked FROM upd;

  -- ── 2. БАЙРШЛЫН ДАВХАРДЛЫГ ДАХИН ТООЦОХ ─────────────────────────────────
  --
  -- repository-гийн DetectParcelOverlaps-тай ЯГ ИЖИЛ тооцоо:
  --   * ST_Overlaps БИШ — тэр нь бүрэн АГУУЛАГДАХ тохиолдлыг false гэдэг,
  --     гэтэл нэг газарт хоёр дугаар нь хамгийн эрсдэлтэй;
  --   * талбайг public.calculate_area_utm-ээр (ГУС-тай ижил UTM) бодно;
  --   * ХОЁР чиглэлээр бичнэ (a→b, b→a).
  DELETE FROM parcel_overlap;

  WITH pairs AS (
    SELECT a.id AS a_id, b.id AS b_id, a.acquisition_id,
           COALESCE(a.parcel_id,'') AS a_code, COALESCE(b.parcel_id,'') AS b_code,
           public.calculate_area_utm(
             ST_Intersection(ST_MakeValid(a.geometry), ST_MakeValid(b.geometry))
           ) AS area_m2
      FROM parcel a
      JOIN parcel b
        ON b.acquisition_id = a.acquisition_id
       AND b.id > a.id
       AND b.deleted_at IS NULL
       AND b.geometry IS NOT NULL
       AND ST_Intersects(a.geometry, b.geometry)
     WHERE a.deleted_at IS NULL
       AND a.geometry IS NOT NULL
  ),
  real_pairs AS (
    SELECT * FROM pairs WHERE area_m2 IS NOT NULL AND area_m2 >= min_overlap_m2
  ),
  ins AS (
    INSERT INTO parcel_overlap (
      parcel_id, other_parcel_id, acquisition_id, other_parcel_code,
      overlap_area_m2, detected_at
    )
    SELECT a_id, b_id, acquisition_id, b_code, area_m2, NOW() FROM real_pairs
    UNION ALL
    SELECT b_id, a_id, acquisition_id, a_code, area_m2, NOW() FROM real_pairs
    ON CONFLICT (parcel_id, other_parcel_id) DO UPDATE SET
      overlap_area_m2   = EXCLUDED.overlap_area_m2,
      other_parcel_code = EXCLUDED.other_parcel_code,
      detected_at       = EXCLUDED.detected_at
    RETURNING parcel_id
  )
  SELECT COUNT(*) INTO overlap_rows FROM ins;

  SELECT COUNT(DISTINCT parcel_id) INTO overlap_parcels FROM parcel_overlap;
  SELECT COUNT(*) INTO without_holder
    FROM parcel p
   WHERE p.deleted_at IS NULL
     AND NOT EXISTS (SELECT 1 FROM parcel_holder ph
                      WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder');

  RAISE NOTICE '── Нэгж талбарын жишээ өгөгдөл ──';
  RAISE NOTICE 'нэгж талбар (нийт)        : %', parcel_total;
  RAISE NOTICE 'жишээ өмчлөгч нэмэгдсэн   : %', holders_inserted;
  RAISE NOTICE 'нэгж талбартай холбогдсон : %', links_inserted;
  RAISE NOTICE 'parcel_detail холбогдсон  : %', details_linked;
  RAISE NOTICE 'эзэмшигчгүй үлдсэн        : %', without_holder;
  RAISE NOTICE 'давхардлын мөр            : %', overlap_rows;
  RAISE NOTICE 'ДАВХАРДАЛТАЙ нэгж талбар  : %', overlap_parcels;
END $$;
