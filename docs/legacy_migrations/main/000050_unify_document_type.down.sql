ALTER TABLE land_acquisition_document ADD COLUMN IF NOT EXISTS file_type INTEGER REFERENCES document_type(id);
