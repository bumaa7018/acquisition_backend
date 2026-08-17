DROP INDEX IF EXISTS idx_decision_draft_current_progress;

ALTER TABLE decision_draft
    DROP COLUMN IF EXISTS current_progress_history_id;
