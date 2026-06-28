DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'parcel_status_history'
      AND column_name = 'parcel_id'
      AND udt_name = 'uuid'
  ) THEN
    ALTER TABLE parcel_status_history
      DROP CONSTRAINT IF EXISTS parcel_status_history_parcel_id_fkey;

    ALTER TABLE parcel_status_history
      ADD COLUMN parcel_code_tmp VARCHAR(12);

    UPDATE parcel_status_history h
    SET parcel_code_tmp = p.parcel_id
    FROM parcel p
    WHERE h.parcel_id = p.id;

    DELETE FROM parcel_status_history
    WHERE parcel_code_tmp IS NULL;

    ALTER TABLE parcel_status_history
      DROP COLUMN parcel_id;

    ALTER TABLE parcel_status_history
      RENAME COLUMN parcel_code_tmp TO parcel_id;

    ALTER TABLE parcel_status_history
      ALTER COLUMN parcel_id SET NOT NULL;

    ALTER TABLE parcel_status_history
      ADD CONSTRAINT parcel_status_history_parcel_id_fkey
      FOREIGN KEY (parcel_id) REFERENCES parcel(parcel_id) ON DELETE CASCADE;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_psh_parcel ON parcel_status_history(parcel_id);

DELETE FROM parcel_status_history
WHERE created_by = 'system-backfill';

INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date, created_by)
SELECT
    p.parcel_id,
    p.acquisition_id,
    s.status_id,
    COALESCE(p.created_at, NOW()) + (s.step_order * INTERVAL '1 day'),
    'system-backfill'
FROM parcel p
JOIN LATERAL (
    VALUES
        (0, 0),
        (1, 1),
        (2, 2),
        (
            CASE
                WHEN p.status IN (3, 4, 5) THEN p.status
                ELSE NULL
            END,
            3
        )
) AS s(status_id, step_order) ON s.status_id IS NOT NULL
JOIN parcel_status ps ON ps.id = s.status_id
WHERE p.status IN (0, 1, 2, 3, 4, 5)
  AND s.step_order <= CASE
      WHEN p.status IN (0, 1, 2) THEN p.status
      ELSE 3
  END
  AND NOT EXISTS (
      SELECT 1
      FROM parcel_status_history h
      WHERE h.parcel_id = p.parcel_id
        AND h.acquisition_id = p.acquisition_id
        AND h.status_id = s.status_id
  );
