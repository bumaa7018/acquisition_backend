WITH managed_menus(code) AS (
  VALUES ('land'), ('compensation'), ('decision'), ('users'), ('roles'), ('permissions'), ('admin'), ('hr'), ('audit')
),
managed_roles(code) AS (
  VALUES ('A100'), ('A101'), ('A102'), ('A103')
)
DELETE FROM sdplatform.sd_role_permission rp
USING sdplatform.sd_role r, sdplatform.sd_menu m, managed_roles mr, managed_menus mm
WHERE rp.role_id = r.role_id
  AND rp.menu_id = m.menu_id
  AND r.code = mr.code
  AND m.code = mm.code;

WITH managed_menus(code) AS (
  VALUES ('land'), ('compensation'), ('decision'), ('users'), ('roles'), ('permissions'), ('admin'), ('hr'), ('audit')
),
managed_roles(code) AS (
  VALUES ('A100'), ('A101'), ('A102'), ('A103')
)
DELETE FROM sdplatform.sd_menu_role mr
USING sdplatform.sd_role r, sdplatform.sd_menu m, managed_roles role_codes, managed_menus mm
WHERE mr.role_id = r.role_id
  AND mr.menu_id = m.menu_id
  AND r.code = role_codes.code
  AND m.code = mm.code;

DELETE FROM sdplatform.sd_user_role ur
USING sdplatform.sd_role r
WHERE ur.role_id = r.role_id
  AND r.code = 'A103';

DELETE FROM sdplatform.sd_system_role sr
USING sdplatform.sd_role r
WHERE sr.role_id = r.role_id
  AND sr.system_id = 10
  AND r.code = 'A103';

DELETE FROM sdplatform.sd_role
WHERE code = 'A103';

DELETE FROM sdplatform.sd_form_permission
WHERE form_code IN ('land', 'compensation', 'decision', 'users', 'roles', 'permissions', 'hr', 'admin', 'audit');
