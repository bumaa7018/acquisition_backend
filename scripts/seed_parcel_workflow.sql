-- ============================================================
-- Seed: нэгж талбарын ажлын урсгал (parcel_workflow)
-- Ажиллуулах: psql "$MAIN_DSN" -f scripts/seed_parcel_workflow.sql
-- Idempotent: UNIQUE(from_status_id, to_status_id) давхардуулахгүй.
-- ============================================================

-- Нэгж талбарын статусуудыг баталгаажуулах
INSERT INTO parcel_status (id, code, name, sort_order) VALUES
    (0, 'pending',      'Хүлээгдэж буй',         0),
    (1, 'processing',   'Зөвшилцөх шатандаа',     1),
    (2, 'evaluating',   'Үнэлгээ хийх',           2),
    (3, 'affected_out', 'Нөлөөлөгдсөн гарсан',    3),
    (4, 'rejected',     'Татгалзсан',             4),
    (5, 'released',     'Чөлөөлсөн',              5)
ON CONFLICT (id) DO UPDATE SET
    code       = EXCLUDED.code,
    name       = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order;

-- Ажлын урсгалын дүрмүүд
-- from_status_id NULL = эхлэх цэг (шинэ нэгж талбар)
INSERT INTO parcel_workflow (from_status_id, to_status_id, sort_order) VALUES
    -- Хүлээгдэж буй → Зөвшилцөх шатандаа
    (0, 1, 1),
    -- Зөвшилцөх шатандаа → Үнэлгээ хийх
    (1, 2, 2),
    -- Зөвшилцөх шатандаа → Нөлөөлөгдсөн гарсан
    (1, 3, 3),
    -- Зөвшилцөх шатандаа → Татгалзсан
    (1, 4, 4),
    -- Үнэлгээ хийх → Чөлөөлсөн
    (2, 5, 1),
    -- Үнэлгээ хийх → Нөлөөлөгдсөн гарсан
    (2, 3, 2),
    -- Үнэлгээ хийх → Татгалзсан
    (2, 4, 3),
    -- Хүлээгдэж буй → Нөлөөлөгдсөн гарсан
    (0, 3, 2)
ON CONFLICT (from_status_id, to_status_id) DO UPDATE SET
    sort_order = EXCLUDED.sort_order;
