ALTER TABLE building
    ADD COLUMN IF NOT EXISTS acquisition_amount NUMERIC(18, 4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN building.acquisition_amount IS 'Барилгын нөхөх олговрын үнэлгээ (₮)';
