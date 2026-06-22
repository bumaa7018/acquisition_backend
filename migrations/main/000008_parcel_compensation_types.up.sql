ALTER TABLE parcel ADD COLUMN IF NOT EXISTS compensation_types TEXT NOT NULL DEFAULT '';
COMMENT ON COLUMN parcel.compensation_types IS 'Нөхөх олговрын төрлүүд (таслалаар тусгаарласан: cash, land, housing, other)';
