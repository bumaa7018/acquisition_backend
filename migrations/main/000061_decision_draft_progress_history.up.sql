CREATE TABLE IF NOT EXISTS decision_draft_progress_history (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_draft_id UUID NOT NULL REFERENCES decision_draft(id) ON DELETE CASCADE,
    progress_type     VARCHAR(30) NOT NULL,
    recipient         TEXT NOT NULL DEFAULT '',
    progress_date     DATE NOT NULL,
    note              TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        TEXT NOT NULL DEFAULT '',
    CONSTRAINT chk_ddph_progress_type
        CHECK (progress_type IN ('reviewing', 'confirming'))
);

CREATE INDEX IF NOT EXISTS idx_ddph_draft_date
    ON decision_draft_progress_history(decision_draft_id, progress_date DESC, created_at DESC);
