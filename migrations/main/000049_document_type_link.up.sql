-- Баримт бичгийг хавсралтын төрөлтэй (document_type) холбоно.
-- Өмнө нь parcel document-ийн document_type_id хаана ч хадгалагддаггүй байсан.
-- (Лавлах утгууд — document_type мөрүүд — seeder-ээр орно, энд зөвхөн schema.)
ALTER TABLE document ADD COLUMN IF NOT EXISTS document_type_id INT REFERENCES document_type(id);
