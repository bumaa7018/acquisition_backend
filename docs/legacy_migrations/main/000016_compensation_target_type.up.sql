-- Replace asset_type with target_type in compensation table.
-- target_type: 'parcel' = нэгж талбарын нөхөн төлбөр
--              'asset'  = хөрөнгийн нөхөн төлбөр (asset_type comes from assets table)

ALTER TABLE compensation ADD COLUMN target_type VARCHAR(20) NOT NULL DEFAULT 'asset';

-- Records that have no linked asset are parcel-level compensation
UPDATE compensation SET target_type = 'parcel' WHERE asset_id IS NULL;

ALTER TABLE compensation DROP COLUMN asset_type;
