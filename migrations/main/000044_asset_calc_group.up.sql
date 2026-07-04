-- Барилгын өртгийн хандлагын үзүүлэлтүүдийг бүлэглэх (Excel-ийн "Итгэлцүүр" мэт
-- row-group header) — asset_calc_type-д бүлгийн нэр нэмнэ.
ALTER TABLE asset_calc_type
  ADD COLUMN IF NOT EXISTS grp TEXT NOT NULL DEFAULT '';

-- Одоо байгаа итгэлцүүрийн төрлүүдийг "Итгэлцүүр" бүлэгт хамааруулна.
UPDATE asset_calc_type SET grp = 'Итгэлцүүр'
WHERE code IN ('coeff_price_increase', 'coeff_interior', 'coeff_casting', 'coeff_brick', 'height');
