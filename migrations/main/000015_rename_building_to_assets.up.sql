DO $$
BEGIN
    IF to_regclass('public.assets') IS NULL AND to_regclass('public.building') IS NOT NULL THEN
        ALTER TABLE building RENAME TO assets;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'assets' AND column_name = 'building_number'
    ) THEN
        ALTER TABLE assets RENAME COLUMN building_number TO asset_number;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'assets' AND column_name = 'building_type'
    ) THEN
        ALTER TABLE assets RENAME COLUMN building_type TO asset_name;
    END IF;
END $$;

ALTER TABLE IF EXISTS assets
    ADD COLUMN IF NOT EXISTS asset_type VARCHAR(20) NOT NULL DEFAULT 'real_state';

UPDATE assets
SET asset_type = 'real_state'
WHERE asset_type = '';

ALTER INDEX IF EXISTS idx_building_acquisition_id RENAME TO idx_assets_acquisition_id;
ALTER INDEX IF EXISTS idx_building_parcel_id RENAME TO idx_assets_parcel_id;
ALTER INDEX IF EXISTS idx_building_geometry RENAME TO idx_assets_geometry;

ALTER TABLE IF EXISTS assets DROP CONSTRAINT IF EXISTS fk_building_parcel_id;

DO $$
BEGIN
    IF to_regclass('public.assets') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conname = 'fk_assets_parcel_id'
             AND conrelid = 'public.assets'::regclass
       ) THEN
        ALTER TABLE assets
            ADD CONSTRAINT fk_assets_parcel_id
            FOREIGN KEY (parcel_id)
            REFERENCES parcel(parcel_id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
            NOT VALID;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'building_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'asset_id'
    ) THEN
        ALTER TABLE compensation RENAME COLUMN building_id TO asset_id;
    END IF;
END $$;

ALTER TABLE IF EXISTS compensation
    DROP CONSTRAINT IF EXISTS compensation_building_id_fkey;

DO $$
BEGIN
    IF to_regclass('public.compensation') IS NOT NULL
       AND to_regclass('public.assets') IS NOT NULL
       AND EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public' AND table_name = 'compensation' AND column_name = 'asset_id'
       )
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conname = 'compensation_asset_id_fkey'
             AND conrelid = 'public.compensation'::regclass
       ) THEN
        ALTER TABLE compensation
            ADD CONSTRAINT compensation_asset_id_fkey
            FOREIGN KEY (asset_id)
            REFERENCES assets(id)
            ON DELETE SET NULL;
    END IF;
END $$;
