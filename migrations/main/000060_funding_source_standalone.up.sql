-- Санхүүгийн эх үүсвэрийг захирамжийн төслөөс ГАРААР шинээр бүртгэх боломж.
--
-- Захирамжийн чөлөөлөлт нь нэгж талбар холбогдох хүртэл тодорхойгүй, харин
-- нэгж талбар холбохын ӨМНӨ эх үүсвэр бүртгэсэн байх шаардлагатай. Тиймээс
-- чөлөөлөлтгүй (бие даасан) эх үүсвэр байхыг зөвшөөрөв.
--
-- Чөлөөлөлтийн "Санхүүжилт" табаас үүсгэсэн мөрүүд урьдын адил
-- acquisition_id-тай хэвээр үлдэнэ.

ALTER TABLE funding_sources ALTER COLUMN acquisition_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_funding_sources_standalone
    ON funding_sources(created_at) WHERE acquisition_id IS NULL;
