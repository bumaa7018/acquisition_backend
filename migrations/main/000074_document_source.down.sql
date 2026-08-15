DROP INDEX IF EXISTS ux_document_source_doc_id;

ALTER TABLE document
    DROP COLUMN IF EXISTS source_doc_id,
    DROP COLUMN IF EXISTS source_ftp_id;
