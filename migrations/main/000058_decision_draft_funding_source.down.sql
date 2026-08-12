ALTER TABLE decision_draft
    ADD COLUMN IF NOT EXISTS funding_source_id UUID REFERENCES funding_sources(id) ON DELETE SET NULL;

-- Олон эх үүсвэрээс хамгийн эрт нэмэгдсэнийг нь буцааж нэг баганад тавина
UPDATE decision_draft d
   SET funding_source_id = (
       SELECT ddfs.funding_source_id
         FROM decision_draft_funding_source ddfs
        WHERE ddfs.decision_draft_id = d.id
        ORDER BY ddfs.created_at
        LIMIT 1
   );

DROP INDEX IF EXISTS idx_ddfs_source;
DROP INDEX IF EXISTS idx_ddfs_draft;
DROP TABLE IF EXISTS decision_draft_funding_source;
