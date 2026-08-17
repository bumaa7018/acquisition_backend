-- Барилгын өртгийн хандлагын мөрүүдийг Excel файл дахь дараалалаар нь харуулахын тулд
-- asset_calculation-д мөрийн дэс дугаар нэмнэ (импортын үед бичигдэнэ).
ALTER TABLE asset_calculation
  ADD COLUMN IF NOT EXISTS sort_index INT NOT NULL DEFAULT 0;
