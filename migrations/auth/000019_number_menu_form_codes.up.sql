-- Цэс болон маягтын кодыг sdplatform-ын A0NNNNN загвараар дугаарлана.
--
-- Өмнө нь цэсний код үгээр ("land", "compensation" ...) байсан. sdplatform-д
-- код нь "A010" + гурван оронтой дугаар байдаг тул үүнд нийцүүлэв:
--
--   A010    — Газар чөлөөлөлт (эцэг)
--   A010001 land          A010006 permissions
--   A010002 compensation  A010007 hr
--   A010003 decision      A010008 admin
--   A010004 users         A010009 audit
--   A010005 roles
--
-- ЧУХАЛ: эрхийн түлхүүр (`land:update`) нь кодоос ХАМААРАХГҮЙ болсон —
-- код ↔ нөөцийн хөрвүүлэлт internal/domain/menucode.go дотор байна. Тиймээс
-- энэ дугаарлалт RequirePermission болон фронтендын шалгалтад НӨЛӨӨЛӨХГҮЙ.

-- 1. Хуучин ашиглагдаагүй A010001..A010005 цэсүүдийг эхлээд зайлуулна
--    (тэдгээр нь sd_form_permission-гүй тул апп аль хэдийн уншдаггүй).
DELETE FROM sdplatform.sd_menu_role
WHERE menu_id IN (
	SELECT m.menu_id FROM sdplatform.sd_menu m
	WHERE m.code IN ('A010001','A010002','A010003','A010004','A010005')
	  AND NOT EXISTS (
		SELECT 1 FROM sdplatform.sd_form_permission fp WHERE fp.form_code = m.code
	  )
);
DELETE FROM sdplatform.sd_menu_form
WHERE menu_code IN ('A010001','A010002','A010003','A010004','A010005')
  AND NOT EXISTS (
	SELECT 1 FROM sdplatform.sd_form_permission fp WHERE fp.form_code = menu_code
  );
DELETE FROM sdplatform.sd_menu m
WHERE m.code IN ('A010001','A010002','A010003','A010004','A010005')
  AND NOT EXISTS (
	SELECT 1 FROM sdplatform.sd_form_permission fp WHERE fp.form_code = m.code
  );

-- 2. Хөрвүүлэлтийн хүснэгт
CREATE TEMP TABLE menu_code_remap ON COMMIT DROP AS
SELECT * FROM (VALUES
	('land',        'A010001'),
	('compensation','A010002'),
	('decision',    'A010003'),
	('users',       'A010004'),
	('roles',       'A010005'),
	('permissions', 'A010006'),
	('hr',          'A010007'),
	('admin',       'A010008'),
	('audit',       'A010009')
) AS t(old_code, new_code);

-- 3. Кодыг агуулдаг бүх хүснэгтийг зэрэг шинэчилнэ
UPDATE sdplatform.sd_menu m
SET code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE m.code = r.old_code;

UPDATE sdplatform.sd_menu m
SET parent_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE m.parent_code = r.old_code;

UPDATE sdplatform.sd_form f
SET code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE f.code = r.old_code;

UPDATE sdplatform.sd_menu_form mf
SET menu_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE mf.menu_code = r.old_code;

UPDATE sdplatform.sd_menu_form mf
SET form_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE mf.form_code = r.old_code;

UPDATE sdplatform.sd_form_permission fp
SET form_code = r.new_code, updated_at = NOW()
FROM menu_code_remap r WHERE fp.form_code = r.old_code;

-- 4. Эцэг цэс A010 — бүх нөөц түүний доор ордог
INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at)
SELECT 'A010', 'Газар чөлөөлөлтийн систем', '/land-acquisitions', 0, NOW()
WHERE NOT EXISTS (SELECT 1 FROM sdplatform.sd_menu WHERE code = 'A010');

UPDATE sdplatform.sd_menu
SET parent_code = 'A010', updated_at = NOW()
WHERE code IN (SELECT new_code FROM menu_code_remap)
  AND parent_code IS DISTINCT FROM 'A010';
