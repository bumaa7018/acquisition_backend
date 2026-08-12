ALTER TABLE decision_draft
    ADD COLUMN IF NOT EXISTS current_progress_history_id UUID
        REFERENCES decision_draft_progress_history(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_decision_draft_current_progress
    ON decision_draft(current_progress_history_id);

WITH latest AS (
    SELECT DISTINCT ON (decision_draft_id)
           id,
           decision_draft_id
      FROM decision_draft_progress_history
     ORDER BY decision_draft_id, progress_date DESC, created_at DESC
)
UPDATE decision_draft d
   SET current_progress_history_id = latest.id
  FROM latest
 WHERE latest.decision_draft_id = d.id
   AND d.current_progress_history_id IS NULL;
