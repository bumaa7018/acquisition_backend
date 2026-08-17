-- Basic new fields for all asset types
ALTER TABLE assets ADD COLUMN IF NOT EXISTS unit        TEXT           NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS capacity    TEXT           NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS description TEXT           NOT NULL DEFAULT '';

-- Building specification fields (real_state type)
ALTER TABLE assets ADD COLUMN IF NOT EXISTS construction_type TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS foundation         TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS walls              TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS roof               TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS door_type          TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS window_type        TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS heating            TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS quality            TEXT NOT NULL DEFAULT '';

-- Cost approach calculation fields (real_state type)
ALTER TABLE assets ADD COLUMN IF NOT EXISTS unit_cost              NUMERIC(18,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS unit_cost_unit         TEXT          NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS coeff_price_increase   NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS coeff_interior         NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS coeff_casting          NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS coeff_brick            NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS height                 NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS height_unit            TEXT          NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS full_replacement_cost  NUMERIC(18,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS depreciation_percent   NUMERIC(10,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS depreciation_amount    NUMERIC(18,4) NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS net_replacement_cost   NUMERIC(18,4) NOT NULL DEFAULT 0;

-- Land valuation table (one record per parcel per acquisition, entered by professional org)
CREATE TABLE IF NOT EXISTS land_valuation (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_id      UUID           NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    parcel_id           VARCHAR(12)    NOT NULL REFERENCES parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE,
    land_area_m2        NUMERIC(18,4)  NOT NULL DEFAULT 0,
    base_price_per_m2   NUMERIC(18,4)  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (acquisition_id, parcel_id)
);

CREATE INDEX IF NOT EXISTS idx_land_valuation_acquisition ON land_valuation(acquisition_id);
CREATE INDEX IF NOT EXISTS idx_land_valuation_parcel      ON land_valuation(parcel_id);
