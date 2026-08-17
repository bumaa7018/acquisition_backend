INSERT INTO decision_draft_progress_history
    (decision_draft_id, progress_type, recipient, progress_date, note, created_at, created_by)
SELECT d.id,
       'confirming',
       '',
       COALESCE(d.decision_date, d.confirmed_at::date, d.updated_at::date, d.created_at::date),
       '',
       COALESCE(d.confirmed_at, d.updated_at, d.created_at),
       COALESCE(d.confirmed_by, '')
  FROM decision_draft d
 WHERE d.status = 2
   AND NOT EXISTS (
       SELECT 1
         FROM decision_draft_progress_history h
        WHERE h.decision_draft_id = d.id
          AND h.progress_type = 'confirming'
   );
