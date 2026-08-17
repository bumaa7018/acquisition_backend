ALTER TABLE compensation ADD COLUMN asset_type VARCHAR(20) NOT NULL DEFAULT 'property';

UPDATE compensation SET asset_type = CASE WHEN target_type = 'parcel' THEN 'property' ELSE 'real_state' END;

ALTER TABLE compensation DROP COLUMN target_type;
