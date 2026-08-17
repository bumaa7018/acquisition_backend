ALTER TABLE parcel_detail
    DROP COLUMN IF EXISTS valuation_zone,
    DROP COLUMN IF EXISTS base_price_per_ha,
    DROP COLUMN IF EXISTS auction_coeff,
    DROP COLUMN IF EXISTS auction_price;
