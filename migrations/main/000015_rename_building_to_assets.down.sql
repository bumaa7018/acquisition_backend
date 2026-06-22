ALTER TABLE IF EXISTS compensation
    DROP CONSTRAINT IF EXISTS compensation_asset_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'asset_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'building_id'
    ) THEN
        ALTER TABLE compensation RENAME COLUMN asset_id TO building_id;
    END IF;
END $$;

DO $$
BEGIN
    IF to_regclass('public.compensation') IS NOT NULL
       AND to_regclass('public.assets') IS NOT NULL
       AND EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'building_id'
       )
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conname = 'compensation_building_id_fkey'
             AND conrelid = 'public.compensation'::regclass
       ) THEN
        ALTER TABLE compensation
            ADD CONSTRAINT compensation_building_id_fkey
            FOREIGN KEY (building_id)
            REFERENCES assets(id)
            ON DELETE SET NULL;
    END IF;
END $$;

ALTER TABLE IF EXISTS assets DROP CONSTRAINT IF EXISTS fk_assets_parcel_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'assets' AND column_name = 'asset_number'
    ) THEN
        ALTER TABLE assets RENAME COLUMN asset_number TO building_number;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'assets' AND column_name = 'asset_name'
    ) THEN
        ALTER TABLE assets RENAME COLUMN asset_name TO building_type;
    END IF;
END $$;

ALTER TABLE IF EXISTS assets
    DROP COLUMN IF EXISTS asset_type;

ALTER INDEX IF EXISTS idx_assets_acquisition_id RENAME TO idx_building_acquisition_id;
ALTER INDEX IF EXISTS idx_assets_parcel_id RENAME TO idx_building_parcel_id;
ALTER INDEX IF EXISTS idx_assets_geometry RENAME TO idx_building_geometry;

DO $$
BEGIN
    IF to_regclass('public.building') IS NULL AND to_regclass('public.assets') IS NOT NULL THEN
        ALTER TABLE assets RENAME TO building;
    END IF;
END $$;
