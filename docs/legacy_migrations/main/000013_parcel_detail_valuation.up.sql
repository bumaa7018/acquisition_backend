ALTER TABLE parcel_detail
    ADD COLUMN IF NOT EXISTS valuation_zone    VARCHAR(200),
    ADD COLUMN IF NOT EXISTS base_price_per_ha NUMERIC(18, 4),
    ADD COLUMN IF NOT EXISTS auction_coeff     NUMERIC(10, 6),
    ADD COLUMN IF NOT EXISTS auction_price     NUMERIC(18, 4);
