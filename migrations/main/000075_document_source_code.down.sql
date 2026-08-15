DROP INDEX IF EXISTS idx_document_source_doc_code;
ALTER TABLE document DROP COLUMN IF EXISTS source_doc_code;
