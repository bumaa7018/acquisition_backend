-- Excel-ээс нөхөн олговрын үнэлгээ импортлоход одоогийн схемд "гэргүй" байсан талбарууд:
--   1) хөрөнгө бүрийн нэгж үнэ (Бусад хөрөнгө хүснэгт)
--   2) газрын өмчлөх эрхийн гэрчилгээ
--   3) үнэлгээ хийсэн байгууллагын мэдээлэл (Sheet1)
--   4) эх Excel файлын нэр/hash — дахин импортоор давхардуулахаас сэргийлэх

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS unit_price NUMERIC(18,4) NOT NULL DEFAULT 0;

ALTER TABLE land_valuation
  ADD COLUMN IF NOT EXISTS ownership_cert_no      TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_org_name     TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_reg_no       TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_state_reg_no TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_director     TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_license      TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_address      TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appraiser_contact      TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS source_file_name       TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS source_file_hash       TEXT NOT NULL DEFAULT '';
