-- Мэргэжлийн байгууллага: чөлөөлөлтөд тэргүүлэх үнэлгээчин
-- users нь authdb-д байдаг тул FK constraint тавих боломжгүй
ALTER TABLE land_acquisition
  ADD COLUMN IF NOT EXISTS professional_org_id UUID;

-- Нэгж талбарт хөндлөнгийн үнэлгээчин
ALTER TABLE parcel
  ADD COLUMN IF NOT EXISTS independent_org_id UUID;

CREATE INDEX IF NOT EXISTS idx_la_professional_org ON land_acquisition(professional_org_id);
CREATE INDEX IF NOT EXISTS idx_parcel_independent_org ON parcel(independent_org_id);
