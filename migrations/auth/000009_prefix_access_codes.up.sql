UPDATE menu_forms
SET form_code = 'A' || form_code
WHERE form_code ~ '^[0-9]' AND form_code NOT LIKE 'A%';

UPDATE menu_forms
SET menu_code = 'A' || menu_code
WHERE menu_code ~ '^[0-9]' AND menu_code NOT LIKE 'A%';

UPDATE form_permissions
SET form_code = 'A' || form_code
WHERE form_code ~ '^[0-9]' AND form_code NOT LIKE 'A%';

UPDATE forms
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE menus
SET parent_code = 'A' || parent_code
WHERE parent_code ~ '^[0-9]' AND parent_code NOT LIKE 'A%';

UPDATE menus
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.sd_menu_form
SET form_code = 'A' || form_code
WHERE form_code ~ '^[0-9]' AND form_code NOT LIKE 'A%';

UPDATE sdplatform.sd_menu_form
SET menu_code = 'A' || menu_code
WHERE menu_code ~ '^[0-9]' AND menu_code NOT LIKE 'A%';

UPDATE sdplatform.sd_form
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.sd_menu
SET parent_code = 'A' || parent_code
WHERE parent_code ~ '^[0-9]' AND parent_code NOT LIKE 'A%';

UPDATE sdplatform.sd_menu
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.sd_permission
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.sd_role
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.hr_position
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

UPDATE sdplatform.hr_department
SET code = 'A' || code
WHERE code ~ '^[0-9]' AND code NOT LIKE 'A%';

