ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS acquisition_amount NUMERIC(18, 4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN assets.acquisition_amount IS 'Хөрөнгийн нөхөх олговрын үнэлгээ (₮)';
