-- land_acquisition_document.file_type нь нэрэндээ хууртсан document_type_id
-- хадгалдаг байсан (INTEGER, FK → document_type) — document.document_type_id-д
-- нэгтгэнэ. Хуучин утгын шилжүүлэг (data) нь seeder алхам
-- 000011_document_type_backfill-д, энд зөвхөн schema өөрчлөлт.
ALTER TABLE land_acquisition_document DROP COLUMN IF EXISTS file_type;
