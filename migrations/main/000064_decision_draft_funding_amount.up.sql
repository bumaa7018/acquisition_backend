ALTER TABLE decision_draft_funding_source
    ADD COLUMN IF NOT EXISTS amount NUMERIC(18,2) NOT NULL DEFAULT 0;

UPDATE decision_draft_funding_source ddfs
   SET amount = COALESCE(fs.amount, 0)
  FROM funding_sources fs
 WHERE fs.id = ddfs.funding_source_id
   AND ddfs.amount = 0;
