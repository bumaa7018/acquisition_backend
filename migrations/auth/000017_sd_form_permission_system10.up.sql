CREATE TABLE IF NOT EXISTS sdplatform.sd_form_permission (
  form_permission_id serial4 NOT NULL,
  form_id int4 NULL,
  permission_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  form_code varchar(20) NULL DEFAULT NULL::character varying,
  permission_code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_form_permission_pkey PRIMARY KEY (form_permission_id)
);

DROP TABLE IF EXISTS tmp_system10_permission_matrix;
CREATE TEMP TABLE tmp_system10_permission_matrix (
  role_code varchar(10) NOT NULL,
  permission_key text NOT NULL
);

INSERT INTO tmp_system10_permission_matrix (role_code, permission_key)
VALUES
  ('A100', 'users:read'), ('A100', 'users:create'), ('A100', 'users:update'), ('A100', 'users:delete'),
  ('A100', 'roles:read'), ('A100', 'roles:create'), ('A100', 'roles:update'), ('A100', 'roles:delete'),
  ('A100', 'permissions:read'), ('A100', 'permissions:create'), ('A100', 'permissions:delete'),
  ('A100', 'admin:read'), ('A100', 'admin:create'), ('A100', 'admin:update'), ('A100', 'admin:delete'),
  ('A100', 'hr:read'), ('A100', 'hr:create'), ('A100', 'hr:structure:create'), ('A100', 'hr:update'), ('A100', 'hr:delete'),
  ('A100', 'audit:read'),
  ('A100', 'land:read'), ('A100', 'land:create'), ('A100', 'land:update'), ('A100', 'land:delete'),
  ('A100', 'compensation:read'), ('A100', 'compensation:create'), ('A100', 'compensation:update'),
  ('A100', 'decision:read'), ('A100', 'decision:create'), ('A100', 'decision:update'), ('A100', 'decision:delete'),

  ('A101', 'users:read'),
  ('A101', 'land:read'), ('A101', 'land:create'), ('A101', 'land:update'), ('A101', 'land:delete'),
  ('A101', 'compensation:read'), ('A101', 'compensation:create'), ('A101', 'compensation:update'),
  ('A101', 'decision:read'), ('A101', 'decision:create'), ('A101', 'decision:update'), ('A101', 'decision:delete'),

  ('A102', 'land:read'), ('A102', 'decision:read'),

  ('A103', 'land:read'),
  ('A103', 'decision:read'), ('A103', 'decision:create'), ('A103', 'decision:update'), ('A103', 'decision:delete');

INSERT INTO sdplatform.sd_system (system_id, "name", description)
VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
ON CONFLICT (system_id) DO UPDATE SET
  "name" = EXCLUDED."name",
  description = EXCLUDED.description;

WITH wanted(code, name, url, sort_order) AS (
  VALUES
    ('land', 'Газар чөлөөлөлт', '/land-acquisitions', 1),
    ('compensation', 'Нөхөх олговор', '/land-acquisitions/:id/compensations', 2),
    ('decision', 'Захирамжийн төсөл', '/decision-drafts', 3),
    ('users', 'Хэрэглэгч', '/users', 10),
    ('roles', 'Хэрэглэгчийн эрх', '/roles', 11),
    ('permissions', 'Permission лавлах', '/roles', 12),
    ('hr', 'Хүний нөөц', '/employee', 13),
    ('admin', 'Системийн тохиргоо', '/acquisition-category', 14),
    ('audit', 'Үйлдлийн лог', '/audit-logs', 15)
)
INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at)
SELECT code, name, url, sort_order, NOW()
FROM wanted
ON CONFLICT (code) DO UPDATE SET
  "name" = EXCLUDED."name",
  menu_url = EXCLUDED.menu_url,
  sort_order = EXCLUDED.sort_order,
  parent_code = NULL,
  updated_at = NOW();

INSERT INTO sdplatform.sd_form (form_name, code, created_at)
SELECT m."name", LEFT(m.code, 20), NOW()
FROM sdplatform.sd_menu m
WHERE m.code IN ('land', 'compensation', 'decision', 'users', 'roles', 'permissions', 'hr', 'admin', 'audit')
  AND NOT EXISTS (SELECT 1 FROM sdplatform.sd_form f WHERE f.code = LEFT(m.code, 20));

INSERT INTO sdplatform.sd_menu_form (form_id, menu_id, form_code, menu_code, created_at)
SELECT f.form_id, m.menu_id, f.code, LEFT(m.code, 20), NOW()
FROM sdplatform.sd_form f
JOIN sdplatform.sd_menu m ON LEFT(m.code, 20) = f.code
WHERE m.code IN ('land', 'compensation', 'decision', 'users', 'roles', 'permissions', 'hr', 'admin', 'audit')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_menu_form mf
    WHERE mf.form_code = f.code AND mf.menu_code = LEFT(m.code, 20)
  );

WITH actions AS (
  SELECT DISTINCT substring(permission_key from position(':' in permission_key) + 1) AS action
  FROM tmp_system10_permission_matrix
)
INSERT INTO sdplatform.sd_permission (code, "name", created_at)
SELECT LEFT(action, 45), LEFT(action, 250), NOW()
FROM actions
WHERE NOT EXISTS (
  SELECT 1 FROM sdplatform.sd_permission p
  WHERE p.name = LEFT(actions.action, 250)
);

WITH split AS (
  SELECT DISTINCT
         split_part(permission_key, ':', 1) AS form_code,
         substring(permission_key from position(':' in permission_key) + 1) AS permission_code
  FROM tmp_system10_permission_matrix
)
INSERT INTO sdplatform.sd_form_permission (form_id, permission_id, form_code, permission_code, created_at, updated_at)
SELECT f.form_id, p.permission_id, f.code, p.name, NOW(), NOW()
FROM split s
JOIN sdplatform.sd_form f ON f.code = s.form_code
JOIN sdplatform.sd_permission p ON p.name = s.permission_code
WHERE NOT EXISTS (
  SELECT 1 FROM sdplatform.sd_form_permission fp
  WHERE fp.form_code = f.code
    AND (fp.permission_id = p.permission_id OR fp.permission_code = p.code OR fp.permission_code = p.name)
);

WITH roles(code, name, description) AS (
  VALUES
    ('A100', 'admin', 'Админ - бүх эрх, тохиргооны удирдлага'),
    ('A101', 'senior_specialist', 'Ахлах мэргэжилтэн - тохиргооноос бусад хэсэгт бүрэн эрхтэй'),
    ('A102', 'employee', 'Мэргэжилтэн - хуучин тохиргооны хязгаарлагдсан ажиллах эрхтэй'),
    ('A103', 'decision_registration', 'Захирамж бүртгэл - захирамж CRUD, чөлөөлөлт/нэгж талбар read-only')
)
INSERT INTO sdplatform.sd_role (code, "name", role_type_code, description, created_at, updated_at)
SELECT code, name, 'system', description, NOW(), NOW()
FROM roles
ON CONFLICT (code) DO UPDATE SET
  "name" = EXCLUDED."name",
  role_type_code = EXCLUDED.role_type_code,
  description = EXCLUDED.description,
  updated_at = NOW();

INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
SELECT 10, r.role_id, NOW()
FROM sdplatform.sd_role r
WHERE r.code IN ('A100', 'A101', 'A102', 'A103')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_system_role sr
    WHERE sr.system_id = 10 AND sr.role_id = r.role_id
  );

WITH old_names(name) AS (
  VALUES
    ('land_admin'), ('compensation_admin'), ('decision_admin'), ('users_admin'), ('roles_admin'),
    ('permissions_admin'), ('admin_admin'), ('hr_admin'), ('audit_admin'),
    ('land_senior_specialist'), ('compensation_senior_specialist'), ('decision_senior_specialist'),
    ('users_senior_specialist'), ('roles_senior_specialist'), ('permissions_senior_specialist'),
    ('admin_senior_specialist'), ('hr_senior_specialist'), ('audit_senior_specialist'),
    ('land_employee'), ('compensation_employee'), ('decision_employee'), ('users_employee'),
    ('roles_employee'), ('permissions_employee'), ('admin_employee'), ('hr_employee'), ('audit_employee'),
    ('land_decision_registration'), ('compensation_decision_registration'), ('decision_decision_registration'),
    ('users_decision_registration'), ('roles_decision_registration'), ('permissions_decision_registration'),
    ('admin_decision_registration'), ('hr_decision_registration'), ('audit_decision_registration')
)
DELETE FROM sdplatform.sd_system_role sr
USING sdplatform.sd_role r, old_names o
WHERE sr.role_id = r.role_id
  AND sr.system_id = 10
  AND r.name = o.name;

WITH old_names(name) AS (
  VALUES
    ('land_admin'), ('compensation_admin'), ('decision_admin'), ('users_admin'), ('roles_admin'),
    ('permissions_admin'), ('admin_admin'), ('hr_admin'), ('audit_admin'),
    ('land_senior_specialist'), ('compensation_senior_specialist'), ('decision_senior_specialist'),
    ('land_employee'), ('compensation_employee'), ('decision_employee')
)
DELETE FROM sdplatform.sd_user_role ur
USING sdplatform.sd_user u, sdplatform.sd_role r, old_names o
WHERE ur.user_id = u.user_id
  AND ur.role_id = r.role_id
  AND lower(u.email) = 'admin@example.com'
  AND r.name = o.name;

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

WITH split AS (
  SELECT role_code,
         split_part(permission_key, ':', 1) AS menu_code
  FROM tmp_system10_permission_matrix
)
INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at, updated_at)
SELECT DISTINCT m.menu_id, r.role_id, NOW(), NOW()
FROM split s
JOIN sdplatform.sd_role r ON r.code = s.role_code
JOIN sdplatform.sd_menu m ON m.code = s.menu_code
ON CONFLICT DO NOTHING;

WITH split AS (
  SELECT role_code,
         split_part(permission_key, ':', 1) AS menu_code,
         substring(permission_key from position(':' in permission_key) + 1) AS permission_code
  FROM tmp_system10_permission_matrix
)
INSERT INTO sdplatform.sd_role_permission (role_id, menu_id, permission_id, created_at, updated_at)
SELECT r.role_id, m.menu_id, p.permission_id, NOW(), NOW()
FROM split s
JOIN sdplatform.sd_role r ON r.code = s.role_code
JOIN sdplatform.sd_menu m ON m.code = s.menu_code
JOIN sdplatform.sd_form f ON f.code = m.code
JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
JOIN sdplatform.sd_permission p ON (
  p.permission_id = fp.permission_id
  OR p.code = fp.permission_code
  OR p.name = fp.permission_code
)
WHERE p.name = s.permission_code
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_role_permission rp
    WHERE rp.role_id = r.role_id
      AND rp.menu_id = m.menu_id
      AND rp.permission_id = p.permission_id
  );

INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
SELECT u.user_id, r.role_id, NOW(), NOW()
FROM sdplatform.sd_user u
JOIN sdplatform.sd_role r ON r.code = 'A100'
WHERE lower(u.email) = 'admin@example.com'
  AND u.deleted_at IS NULL
ON CONFLICT DO NOTHING;

DROP TABLE IF EXISTS tmp_system10_permission_matrix;
