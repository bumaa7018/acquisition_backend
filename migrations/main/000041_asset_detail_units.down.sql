DROP TABLE IF EXISTS asset_calculation;
DROP TABLE IF EXISTS asset_calc_type;
DROP TABLE IF EXISTS asset_spec;
DROP TABLE IF EXISTS asset_spec_type;

-- Recreate asset_detail (reverts to migration 000040 state)
CREATE TABLE IF NOT EXISTS asset_detail (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id             UUID          NOT NULL UNIQUE REFERENCES assets(id) ON DELETE CASCADE,
    construction_type    TEXT          NOT NULL DEFAULT '',
    foundation           TEXT          NOT NULL DEFAULT '',
    walls                TEXT          NOT NULL DEFAULT '',
    roof                 TEXT          NOT NULL DEFAULT '',
    door_type            TEXT          NOT NULL DEFAULT '',
    window_type          TEXT          NOT NULL DEFAULT '',
    heating              TEXT          NOT NULL DEFAULT '',
    quality              TEXT          NOT NULL DEFAULT '',
    unit_cost            NUMERIC(18,4) NOT NULL DEFAULT 0,
    unit_cost_unit       TEXT          NOT NULL DEFAULT '',
    coeff_price_increase NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_interior       NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_casting        NUMERIC(10,4) NOT NULL DEFAULT 0,
    coeff_brick          NUMERIC(10,4) NOT NULL DEFAULT 0,
    height               NUMERIC(10,4) NOT NULL DEFAULT 0,
    height_unit          TEXT          NOT NULL DEFAULT '',
    full_replacement_cost  NUMERIC(18,4) NOT NULL DEFAULT 0,
    depreciation_percent   NUMERIC(10,4) NOT NULL DEFAULT 0,
    depreciation_amount    NUMERIC(18,4) NOT NULL DEFAULT 0,
    net_replacement_cost   NUMERIC(18,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
