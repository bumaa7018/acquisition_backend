-- «Нийслэл» төсөв (санхүүжилтийн төрөл) — хуучин мэдээллийн импортод
-- санхүүжилтийн төрөл тодорхойгүй захирамжийн өгөгдмөл утга.
-- Тохиргооноос ижил нэртэй сонголт аль хэдийн нэмсэн бол давхар үүсгэхгүй.
INSERT INTO public.decision_budget (code, name, sort_order)
SELECT 'capital', 'Нийслэл', COALESCE((SELECT MAX(sort_order) FROM public.decision_budget), 0) + 1
 WHERE NOT EXISTS (SELECT 1 FROM public.decision_budget WHERE lower(btrim(name)) = 'нийслэл' OR code = 'capital');
