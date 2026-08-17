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

CREATE TABLE IF NOT EXISTS sdplatform.sd_form (
  form_id serial4 NOT NULL,
  form_name varchar(250) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_form_pkey PRIMARY KEY (form_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_menu_form (
  menu_form_id serial4 NOT NULL,
  form_id int4 NULL,
  menu_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  form_code varchar(20) NULL DEFAULT NULL::character varying,
  menu_code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_menu_form_pkey PRIMARY KEY (menu_form_id)
);

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

WITH wanted(key) AS (
  VALUES
    ('users:read'), ('users:create'), ('users:update'), ('users:delete'),
    ('roles:read'), ('roles:create'), ('roles:update'), ('roles:delete'),
    ('permissions:read'), ('permissions:create'), ('permissions:delete'),
    ('admin:read'), ('admin:create'), ('admin:update'), ('admin:delete'),
    ('hr:read'), ('hr:create'), ('hr:structure:create'), ('hr:update'), ('hr:delete'),
    ('audit:read'),
    ('land:read'), ('land:create'), ('land:update'), ('land:delete'),
    ('compensation:read'), ('compensation:create'), ('compensation:update'),
    ('decision:read'), ('decision:create'), ('decision:update'), ('decision:delete')
),
actions AS (
  SELECT DISTINCT substring(key from position(':' in key) + 1) AS action
  FROM wanted
)
INSERT INTO sdplatform.sd_permission (code, "name", created_at)
SELECT LEFT(action, 45), LEFT(action, 250), NOW()
FROM actions
WHERE NOT EXISTS (
  SELECT 1 FROM sdplatform.sd_permission p
  WHERE p.name = LEFT(actions.action, 250)
);

WITH wanted(key) AS (
  VALUES
    ('users:read'), ('users:create'), ('users:update'), ('users:delete'),
    ('roles:read'), ('roles:create'), ('roles:update'), ('roles:delete'),
    ('permissions:read'), ('permissions:create'), ('permissions:delete'),
    ('admin:read'), ('admin:create'), ('admin:update'), ('admin:delete'),
    ('hr:read'), ('hr:create'), ('hr:structure:create'), ('hr:update'), ('hr:delete'),
    ('audit:read'),
    ('land:read'), ('land:create'), ('land:update'), ('land:delete'),
    ('compensation:read'), ('compensation:create'), ('compensation:update'),
    ('decision:read'), ('decision:create'), ('decision:update'), ('decision:delete')
),
split AS (
  SELECT split_part(key, ':', 1) AS form_code,
         substring(key from position(':' in key) + 1) AS permission_code
  FROM wanted
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

WITH resources(code) AS (
  VALUES ('compensation')
),
levels(level_name, description) AS (
  VALUES
    ('professional_org', 'Мэргэжлийн байгууллага'),
    ('mika', 'МИКА'),
    ('finance_specialist', 'Санхүүгийн мэргэжилтэн')
)
INSERT INTO sdplatform.sd_role ("name", role_type_code, description, created_at, updated_at)
SELECT resources.code || '_' || levels.level_name, resources.code, levels.description || ' — ' || resources.code, NOW(), NOW()
FROM resources CROSS JOIN levels
WHERE NOT EXISTS (
  SELECT 1 FROM sdplatform.sd_role r
  WHERE r.name = resources.code || '_' || levels.level_name
);

INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
SELECT 10, r.role_id, NOW()
FROM sdplatform.sd_role r
WHERE r.role_type_code IN ('compensation')
  AND r.name IN ('compensation_professional_org', 'compensation_mika', 'compensation_finance_specialist')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_system_role sr
    WHERE sr.system_id = 10 AND sr.role_id = r.role_id
  );

INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at, updated_at)
SELECT r.role_id, p.permission_id, m.menu_id, NOW(), NOW()
FROM sdplatform.sd_role r
JOIN sdplatform.sd_menu m ON m.code = 'compensation'
JOIN sdplatform.sd_permission p ON p.name IN ('read', 'create', 'update')
WHERE r.name IN ('compensation_professional_org', 'compensation_mika')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id AND rp.menu_id = m.menu_id
  );

INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at, updated_at)
SELECT r.role_id, p.permission_id, m.menu_id, NOW(), NOW()
FROM sdplatform.sd_role r
JOIN sdplatform.sd_menu m ON m.code = 'compensation'
JOIN sdplatform.sd_permission p ON p.name IN ('read', 'update')
WHERE r.name = 'compensation_finance_specialist'
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id AND rp.menu_id = m.menu_id
  );
