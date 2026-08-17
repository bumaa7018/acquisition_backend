UPDATE compensation
SET asset_type = CASE asset_type
    WHEN 'building' THEN 'real_state'
    WHEN 'parcel' THEN 'property'
    ELSE asset_type
END
WHERE asset_type IN ('building', 'parcel');

ALTER TABLE compensation
    ALTER COLUMN asset_type SET DEFAULT 'property';
