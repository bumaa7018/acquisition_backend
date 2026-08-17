ALTER TABLE parcel ADD COLUMN IF NOT EXISTS acquisition_amount NUMERIC(18, 4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN parcel.acquisition_amount IS 'Нөхөх олговрын үнэлгээ (₮)';
