UPDATE compensation
SET asset_type = CASE asset_type
    WHEN 'real_state' THEN 'building'
    WHEN 'property' THEN 'parcel'
    ELSE asset_type
END
WHERE asset_type IN ('real_state', 'property');

ALTER TABLE compensation
    ALTER COLUMN asset_type SET DEFAULT 'parcel';
