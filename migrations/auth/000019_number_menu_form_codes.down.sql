-- A0NNNNN дугаарлалтыг үгэн код руу буцаана.
CREATE TEMP TABLE menu_code_unmap ON COMMIT DROP AS
SELECT * FROM (VALUES
	('A010001','land'),
	('A010002','compensation'),
	('A010003','decision'),
	('A010004','users'),
	('A010005','roles'),
	('A010006','permissions'),
	('A010007','hr'),
	('A010008','admin'),
	('A010009','audit')
) AS t(new_code, old_code);

UPDATE sdplatform.sd_menu SET parent_code = NULL, updated_at = NOW()
WHERE parent_code = 'A010';

UPDATE sdplatform.sd_form_permission fp
SET form_code = u.old_code, updated_at = NOW()
FROM menu_code_unmap u WHERE fp.form_code = u.new_code;

UPDATE sdplatform.sd_menu_form mf
SET form_code = u.old_code, updated_at = NOW()
FROM menu_code_unmap u WHERE mf.form_code = u.new_code;

UPDATE sdplatform.sd_menu_form mf
SET menu_code = u.old_code, updated_at = NOW()
FROM menu_code_unmap u WHERE mf.menu_code = u.new_code;

UPDATE sdplatform.sd_form f
SET code = u.old_code, updated_at = NOW()
FROM menu_code_unmap u WHERE f.code = u.new_code;

UPDATE sdplatform.sd_menu m
SET code = u.old_code, updated_at = NOW()
FROM menu_code_unmap u WHERE m.code = u.new_code;

DELETE FROM sdplatform.sd_menu WHERE code = 'A010';
