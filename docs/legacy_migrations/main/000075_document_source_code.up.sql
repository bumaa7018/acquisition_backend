-- ГУС-ийн баримтын ҮҮРГИЙН код (doc_code). Урьд нь зөвхөн файлын нэр гаргахад
-- ашиглагдаад хаягддаг байсан. Хавсралтыг дэлгэц дээр Кадастр/Төлбөр/
-- Мониторинг гэж бүлэглэхэд шаардлагатай тул хадгална.
--
-- ЗӨВХӨН ГУС-аас татагдсан баримтад утгатай — гараар оруулсан дээр NULL.
-- Манай document_type.id-тай ХОЛИХГҮЙ: дугаарлалт нь огт өөр.
ALTER TABLE document
    ADD COLUMN IF NOT EXISTS source_doc_code VARCHAR(20);

COMMENT ON COLUMN document.source_doc_code IS 'ГУС-ийн баримтын үүргийн код (ж: 1=Өргөдөл, 11=Кадастрын зураг). document_type_id-тай хамааралгүй.';

CREATE INDEX IF NOT EXISTS idx_document_source_doc_code
    ON document (source_doc_code) WHERE source_doc_code IS NOT NULL;
