-- ГАРААР ажиллуулах: цэс/маягтын үгэн кодыг A0NNNNN рүү шилжүүлнэ.
-- ЗӨВХӨН аппын өөрийн локал/dev DB дээр ажиллуул. Жинхэнэ sdplatform DB дээр
-- ажиллуулбал өөр системийн мөрийг өөрчлөх эрсдэлтэй.
BEGIN;
CREATE TEMP TABLE menu_code_remap ON COMMIT DROP AS
SELECT * FROM (VALUES
	('land','A010101'), ('compensation','A010102'), ('decision','A010103'),
	('users','A010104'), ('roles','A010105'), ('permissions','A010106'),
	('hr','A010107'), ('admin','A010108'), ('audit','A010109')
) AS t(old_code, new_code);

UPDATE sdplatform.sd_menu m SET code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE m.code = r.old_code;
UPDATE sdplatform.sd_menu m SET parent_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE m.parent_code = r.old_code;
UPDATE sdplatform.sd_form f SET code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE f.code = r.old_code;
UPDATE sdplatform.sd_menu_form mf SET menu_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE mf.menu_code = r.old_code;
UPDATE sdplatform.sd_menu_form mf SET form_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE mf.form_code = r.old_code;
UPDATE sdplatform.sd_form_permission fp SET form_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE fp.form_code = r.old_code;
COMMIT;
