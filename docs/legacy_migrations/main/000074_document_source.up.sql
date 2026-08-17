-- Эх системээс (дундын сервисийн /application/documents) татагдсан хавсралт.
--
-- Хавсралт татах нь ЗӨВХӨН НЭМЭХ үйлдэл: эх системд байгаа боловч энд
-- байхгүй баримт л шинээр бүртгэгдэнэ. Аль баримт аль хэдийн татагдсаныг
-- source_doc_id (ct_document.id)-ээр таньдаг тул давхардал үүсэхгүй.
ALTER TABLE document
    ADD COLUMN IF NOT EXISTS source_doc_id TEXT,
    ADD COLUMN IF NOT EXISTS source_ftp_id TEXT;

COMMENT ON COLUMN document.source_doc_id IS 'Эх системийн баримтын дугаар (ct_document.id). Гараар оруулсан хавсралт дээр NULL';
COMMENT ON COLUMN document.source_ftp_id IS 'Эх системийн файл серверийн дугаар (ct_document.ftp_id)';

-- Нэг эх баримт НЭГ л удаа бүртгэгдэнэ. Хэсэгчилсэн index: гараар оруулсан
-- (source_doc_id IS NULL) хавсралтууд хязгаарлагдахгүй.
CREATE UNIQUE INDEX IF NOT EXISTS ux_document_source_doc_id
    ON document (source_doc_id)
    WHERE source_doc_id IS NOT NULL;
