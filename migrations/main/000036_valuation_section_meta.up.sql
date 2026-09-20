-- Үнэлгээний хүснэгтийн ДУГААР, НЭР, ДАРААЛАЛ.
--
-- ЯАГААД: үнэлгээний загварын хүснэгт бүр өөрийн дугаар ("3.3"), шошго
-- ("Хүснэгт-3") ба гарчигтай ("Газрын үнэлгээ"). Дэлгэц дээр хүснэгтүүдийг ЯГ
-- Excel-ийнх шиг дугаарлаж, мөн адил дарааллаар харуулах шаардлагатай. Загвар
-- шинэчлэгдэж дугаарлалт/дараалал өөрчлөгдвөл (жишээ нь шинэ хүснэгт нэмэгдэх)
-- дэлгэц нь кодын хатуу дарааллаар биш, ОРУУЛСАН ФАЙЛЫН дарааллаар гарах ёстой.
--
-- Эдгээр нь valuation_section_note мөр дээр хадгалагдана: нэг хэсэг = нэг мөр
-- (тайлбар + таних мэдээлэл). Тайлбаргүй ч хүснэгтийн мета мэдээлэл хадгалагдаж
-- болно (note = '').
ALTER TABLE public.valuation_section_note
    ADD COLUMN IF NOT EXISTS section_no text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS table_label text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS title text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS sort_order integer DEFAULT 0 NOT NULL;

COMMENT ON COLUMN public.valuation_section_note.section_no IS
    'Excel дэх хэсгийн дугаар ("3.3"). Хоосон бол загварын өгөгдмөл дараалал хэрэглэнэ.';
COMMENT ON COLUMN public.valuation_section_note.table_label IS
    'Excel дэх хүснэгтийн шошго ("Хүснэгт-3").';
COMMENT ON COLUMN public.valuation_section_note.title IS
    'Хүснэгтийн гарчиг дугааргүйгээр ("Газрын үнэлгээ").';
COMMENT ON COLUMN public.valuation_section_note.sort_order IS
    'Оруулсан файл дахь хүснэгтийн дараалал (0-оос). Дэлгэц энэ дарааллаар харуулна.';
