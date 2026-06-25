CREATE TABLE IF NOT EXISTS land_acquisition_document (
    land_acquisition_id UUID    NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    document_id         UUID    NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    file_type           INTEGER NOT NULL REFERENCES document_type(id),
    PRIMARY KEY (land_acquisition_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_la_doc_acquisition ON land_acquisition_document (land_acquisition_id);
