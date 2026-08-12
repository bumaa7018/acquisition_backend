DROP INDEX IF EXISTS idx_funding_sources_standalone;

-- NOT NULL-ыг сэргээхийн өмнө чөлөөлөлтгүй мөрүүдийг устгана (өөр утга
-- оноох боломжгүй — тэдгээр нь ямар ч чөлөөлөлтөд харьяалагдахгүй).
DELETE FROM funding_sources WHERE acquisition_id IS NULL;

ALTER TABLE funding_sources ALTER COLUMN acquisition_id SET NOT NULL;
