DROP TABLE IF EXISTS compensation_history;
ALTER TABLE compensation
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS review_note,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS reviewed_at;
