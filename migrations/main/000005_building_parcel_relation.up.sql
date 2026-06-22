UPDATE assets a
SET parcel_id = p.parcel_id
FROM parcel p
WHERE a.parcel_id = p.id::text;

ALTER TABLE assets
    DROP CONSTRAINT IF EXISTS fk_assets_parcel_id;

ALTER TABLE assets
    ADD CONSTRAINT fk_assets_parcel_id
    FOREIGN KEY (parcel_id)
    REFERENCES parcel(parcel_id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
    NOT VALID;
