-- Multi-stream үнэлгээ: нэгж талбар бүрт asset/independent/mika гэсэн 3 тусдаа
-- үнэлгээний урсгал. Урсгал бүр өөрийн хөрөнгө/олговор/газрын үнэлгээ/илгээх төлөвтэй
-- бөгөөд өгөгдөл нь хоорондоо холилдохгүй. Санхүү зөвшөөрсөн урсгал нь тухайн
-- нэгж талбарын "үндсэн" (selected_valuation_type) болж, цаашид ашиглагдана.

ALTER TABLE assets                              ADD COLUMN IF NOT EXISTS valuation_type TEXT NOT NULL DEFAULT 'asset';
ALTER TABLE compensation                        ADD COLUMN IF NOT EXISTS valuation_type TEXT NOT NULL DEFAULT 'asset';
ALTER TABLE land_valuation                      ADD COLUMN IF NOT EXISTS valuation_type TEXT NOT NULL DEFAULT 'asset';
ALTER TABLE parcel_valuation_submission         ADD COLUMN IF NOT EXISTS valuation_type TEXT NOT NULL DEFAULT 'asset';
ALTER TABLE parcel_valuation_submission_history ADD COLUMN IF NOT EXISTS valuation_type TEXT NOT NULL DEFAULT 'asset';
ALTER TABLE parcel                              ADD COLUMN IF NOT EXISTS selected_valuation_type TEXT;

-- unique key-үүдэд valuation_type нэмнэ (урсгал бүр өөрийн мөртэй)
ALTER TABLE land_valuation DROP CONSTRAINT IF EXISTS land_valuation_acquisition_id_parcel_id_key;
ALTER TABLE land_valuation ADD CONSTRAINT land_valuation_acq_parcel_type_key
    UNIQUE (acquisition_id, parcel_id, valuation_type);

ALTER TABLE parcel_valuation_submission DROP CONSTRAINT IF EXISTS parcel_valuation_submission_acquisition_id_parcel_id_key;
ALTER TABLE parcel_valuation_submission ADD CONSTRAINT pvs_acq_parcel_type_key
    UNIQUE (acquisition_id, parcel_id, valuation_type);

CREATE INDEX IF NOT EXISTS idx_assets_valuation_type ON assets (valuation_type);
CREATE INDEX IF NOT EXISTS idx_compensation_valuation_type ON compensation (valuation_type);

-- Backfill (зөвшөөрөгдсөн парцелийг asset болгох) нь seeder алхам
-- 000010_selected_valuation_backfill-д хийгдэнэ — migration зөвхөн schema.
