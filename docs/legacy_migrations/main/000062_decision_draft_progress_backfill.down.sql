DELETE FROM decision_draft_progress_history h
 WHERE h.progress_type = 'confirming'
   AND h.recipient = ''
   AND h.note = ''
   AND EXISTS (
       SELECT 1
         FROM decision_draft d
        WHERE d.id = h.decision_draft_id
          AND d.status = 2
          AND COALESCE(d.confirmed_by, '') = h.created_by
          AND COALESCE(d.decision_date, d.confirmed_at::date, d.updated_at::date, d.created_at::date) = h.progress_date
   );
