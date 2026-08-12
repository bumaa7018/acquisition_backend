-- Нэгж талбар бүрийг ЯМАР санхүүгийн эх үүсвэрээр санхүүжүүлэхийг холбох
-- үедээ сонгоно (дараа нь солих боломжтой).
--
-- decision_draft_funding_source руу заана — өөрөөр хэлбэл зөвхөн ТУХАЙН
-- захирамжид нэмэгдсэн эх үүсвэрээс сонгоно. ON DELETE RESTRICT нь
-- ашиглагдаж буй эх үүсвэрийг захирамжаас санамсаргүй хасахаас хамгаална
-- (service давхарга үүнийг урьдчилан ойлгомжтой алдаагаар шалгана).

ALTER TABLE decision_draft_parcel
    ADD COLUMN IF NOT EXISTS funding_link_id UUID
        REFERENCES decision_draft_funding_source(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_ddp_funding_link
    ON decision_draft_parcel(funding_link_id);
