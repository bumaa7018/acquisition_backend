UPDATE building b
SET parcel_id = p.parcel_id
FROM parcel p
WHERE b.parcel_id = p.id::text;

ALTER TABLE building
    DROP CONSTRAINT IF EXISTS fk_building_parcel_id;

ALTER TABLE building
    ADD CONSTRAINT fk_building_parcel_id
    FOREIGN KEY (parcel_id)
    REFERENCES parcel(parcel_id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
    NOT VALID;
