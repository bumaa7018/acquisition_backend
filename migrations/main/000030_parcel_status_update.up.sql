-- 1. Remove workflow rules referencing statuses outside 0-5
DELETE FROM parcel_workflow
WHERE (from_status_id IS NOT NULL AND from_status_id NOT IN (0,1,2,3,4,5))
   OR to_status_id NOT IN (0,1,2,3,4,5);

-- 2. Reset parcel.status to 0 (pending) for any rows referencing removed statuses
UPDATE parcel SET status = 0 WHERE status NOT IN (0,1,2,3,4,5);

-- 3. Delete all rows not in the canonical 6
DELETE FROM parcel_status WHERE id NOT IN (0,1,2,3,4,5);

-- 4. Upsert the 6 canonical statuses
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

-- 5. New parcels start at 0 (pending) instead of 5 (released)
ALTER TABLE parcel ALTER COLUMN status SET DEFAULT 0;
