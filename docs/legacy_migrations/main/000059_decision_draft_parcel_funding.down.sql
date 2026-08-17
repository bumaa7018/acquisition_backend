DROP INDEX IF EXISTS idx_ddp_funding_link;

ALTER TABLE decision_draft_parcel DROP COLUMN IF EXISTS funding_link_id;
