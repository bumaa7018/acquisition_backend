-- ILIKE '%...%' (leading wildcard) гэж btree index ашигладаггүй.
-- pg_trgm + GIN index ашиглан trigram-аар хайлт хийнэ.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- land_acquisition: хайлтад ашиглагдах боловч index байхгүй column-ууд
CREATE INDEX IF NOT EXISTS idx_land_acquisition_start_date        ON land_acquisition (start_date);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_created_at        ON land_acquisition (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_construction_type ON land_acquisition (construction_type_id);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_decree_date       ON land_acquisition (decree_date);

-- ListReportParcels-д EXTRACT(YEAR FROM COALESCE(decree_date, start_date)) = $n хэлбэртэй filter ашиглана
CREATE INDEX IF NOT EXISTS idx_land_acquisition_decree_year
    ON land_acquisition (EXTRACT(YEAR FROM COALESCE(decree_date, start_date)));

-- plan_code болон acquisition_name ILIKE '%...%' хайлт
CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_code_trgm
    ON land_acquisition USING GIN (plan_code gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_acq_name_trgm
    ON land_acquisition USING GIN (acquisition_name gin_trgm_ops);

-- parcel: хайлтад ашиглагдах боловч index байхгүй column-ууд
CREATE INDEX IF NOT EXISTS idx_parcel_right_type ON parcel (right_type);
CREATE INDEX IF NOT EXISTS idx_parcel_landuse    ON parcel (landuse);
CREATE INDEX IF NOT EXISTS idx_parcel_created_at ON parcel (created_at DESC);

-- parcel_id ILIKE '%...%' хайлт (UNIQUE btree нь leading % сканд ашиглагдахгүй)
CREATE INDEX IF NOT EXISTS idx_parcel_parcel_id_trgm
    ON parcel USING GIN (parcel_id gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_parcel_landuse_trgm
    ON parcel USING GIN (landuse gin_trgm_ops);

-- compensation: JOIN болон EXISTS subquery-д ашиглагдах column-ууд
-- (acquisition_id, parcel_id, compensation_type) — EXISTS дотор гурвуулаа давхар тохируулагддаг
CREATE INDEX IF NOT EXISTS idx_compensation_acq_parcel_type
    ON compensation (acquisition_id, parcel_id, compensation_type);
CREATE INDEX IF NOT EXISTS idx_compensation_asset_id
    ON compensation (asset_id) WHERE asset_id IS NOT NULL;

-- compensation_grant: acquisition-аар хайх
CREATE INDEX IF NOT EXISTS idx_compensation_grant_acquisition
    ON compensation_grant (acquisition_id);
