DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'compensation'
          AND column_name = 'parcel_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'compensation'
          AND column_name = 'parcel_code'
    ) THEN
        ALTER TABLE compensation RENAME COLUMN parcel_id TO parcel_code;
    END IF;
END $$;
