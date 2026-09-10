-- TEXT → BOOLEAN. Тайлбарын АГУУЛГА алдагдана (хоосон биш бүхэн = true).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'parcel'
           AND column_name = 'db_changed'
           AND data_type = 'text'
    ) THEN
        ALTER TABLE parcel ALTER COLUMN db_changed DROP DEFAULT;
        ALTER TABLE parcel
            ALTER COLUMN db_changed TYPE boolean
            USING (COALESCE(NULLIF(btrim(db_changed), ''), '') <> '');
        ALTER TABLE parcel ALTER COLUMN db_changed SET DEFAULT false;
        ALTER TABLE parcel ALTER COLUMN db_changed SET NOT NULL;
    END IF;
END $$;
