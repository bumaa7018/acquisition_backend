-- Remove incorrectly placed columns from assets table (added in 000039)
ALTER TABLE assets DROP COLUMN IF EXISTS construction_type;
ALTER TABLE assets DROP COLUMN IF EXISTS foundation;
ALTER TABLE assets DROP COLUMN IF EXISTS walls;
ALTER TABLE assets DROP COLUMN IF EXISTS roof;
ALTER TABLE assets DROP COLUMN IF EXISTS door_type;
ALTER TABLE assets DROP COLUMN IF EXISTS window_type;
ALTER TABLE assets DROP COLUMN IF EXISTS heating;
ALTER TABLE assets DROP COLUMN IF EXISTS quality;
ALTER TABLE assets DROP COLUMN IF EXISTS unit_cost;
ALTER TABLE assets DROP COLUMN IF EXISTS unit_cost_unit;
ALTER TABLE assets DROP COLUMN IF EXISTS coeff_price_increase;
ALTER TABLE assets DROP COLUMN IF EXISTS coeff_interior;
ALTER TABLE assets DROP COLUMN IF EXISTS coeff_casting;
ALTER TABLE assets DROP COLUMN IF EXISTS coeff_brick;
ALTER TABLE assets DROP COLUMN IF EXISTS height;
ALTER TABLE assets DROP COLUMN IF EXISTS height_unit;
ALTER TABLE assets DROP COLUMN IF EXISTS full_replacement_cost;
ALTER TABLE assets DROP COLUMN IF EXISTS depreciation_percent;
ALTER TABLE assets DROP COLUMN IF EXISTS depreciation_amount;
ALTER TABLE assets DROP COLUMN IF EXISTS net_replacement_cost;

-- Proper asset_detail table: one-to-one with assets, holds building spec & cost approach data
CREATE TABLE IF NOT EXISTS asset_detail (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id              UUID          NOT NULL UNIQUE REFERENCES assets(id) ON DELETE CASCADE,
    -- Барилгын үзүүлэлт (real_state)
    construction_type     TEXT          NOT NULL DEFAULT '',
    foundation            TEXT          NOT NULL DEFAULT '',
    walls                 TEXT          NOT NULL DEFAULT '',
    roof                  TEXT          NOT NULL DEFAULT '',
    door_type             TEXT          NOT NULL DEFAULT '',
    window_type           TEXT          NOT NULL DEFAULT '',
    heating               TEXT          NOT NULL DEFAULT '',
    quality               TEXT          NOT NULL DEFAULT '',
    -- Байгааламжийн өртгийн хандлагаарх тооцоолол
    unit_cost             NUMERIC(18,4) NOT NULL DEFAULT 0,
    unit_cost_unit        TEXT          NOT NULL DEFAULT '',
    coeff_price_increase  NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_interior        NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_casting         NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_brick           NUMERIC(10,4) NOT NULL DEFAULT 0,
    height                NUMERIC(10,4) NOT NULL DEFAULT 0,
    height_unit           TEXT          NOT NULL DEFAULT '',
    full_replacement_cost NUMERIC(18,4) NOT NULL DEFAULT 0,
    depreciation_percent  NUMERIC(10,4) NOT NULL DEFAULT 0,
    depreciation_amount   NUMERIC(18,4) NOT NULL DEFAULT 0,
    net_replacement_cost  NUMERIC(18,4) NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_asset_detail_asset_id ON asset_detail(asset_id);
