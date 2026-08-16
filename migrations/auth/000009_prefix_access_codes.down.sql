UPDATE menu_forms
SET form_code = substring(form_code from 2)
WHERE form_code ~ '^A[0-9]';

UPDATE menu_forms
SET menu_code = substring(menu_code from 2)
WHERE menu_code ~ '^A[0-9]';

UPDATE form_permissions
SET form_code = substring(form_code from 2)
WHERE form_code ~ '^A[0-9]';

UPDATE forms
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE menus
SET parent_code = substring(parent_code from 2)
WHERE parent_code ~ '^A[0-9]';

UPDATE menus
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.sd_menu_form
SET form_code = substring(form_code from 2)
WHERE form_code ~ '^A[0-9]';

UPDATE sdplatform.sd_menu_form
SET menu_code = substring(menu_code from 2)
WHERE menu_code ~ '^A[0-9]';

UPDATE sdplatform.sd_form
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.sd_menu
SET parent_code = substring(parent_code from 2)
WHERE parent_code ~ '^A[0-9]';

UPDATE sdplatform.sd_menu
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.sd_permission
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.sd_role
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.hr_position
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';

UPDATE sdplatform.hr_department
SET code = substring(code from 2)
WHERE code ~ '^A[0-9]';
