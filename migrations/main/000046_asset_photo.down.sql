ALTER TABLE assets
  DROP COLUMN IF EXISTS photo_pdf_url,
  DROP COLUMN IF EXISTS photo_pdf_name;
