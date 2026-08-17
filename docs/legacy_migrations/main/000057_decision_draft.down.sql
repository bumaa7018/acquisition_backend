DROP INDEX IF EXISTS uq_ddp_active_parcel;
DROP INDEX IF EXISTS idx_ddp_parcel;
DROP INDEX IF EXISTS idx_ddp_draft;
DROP TABLE IF EXISTS decision_draft_parcel;

DROP INDEX IF EXISTS idx_decision_draft_proposal_no;
DROP INDEX IF EXISTS idx_decision_draft_decree_number;
DROP INDEX IF EXISTS idx_decision_draft_acquisition;
DROP INDEX IF EXISTS idx_decision_draft_status;
DROP TABLE IF EXISTS decision_draft;

DROP TABLE IF EXISTS decision_budget;
DROP TABLE IF EXISTS decision_work_type;
