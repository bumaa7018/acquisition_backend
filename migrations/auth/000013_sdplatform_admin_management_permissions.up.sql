INSERT INTO sdplatform.sd_permission (code, name, created_at)
SELECT v.code, v.name, NOW()
FROM (VALUES
  ('A012', 'users:read'),
  ('A013', 'users:create'),
  ('A014', 'roles:read'),
  ('A015', 'permissions:read'),
  ('A016', 'hr:read'),
  ('A017', 'hr:create')
) AS v(code, name)
WHERE NOT EXISTS (
  SELECT 1 FROM sdplatform.sd_permission p
  WHERE p.code = v.code OR p.name = v.name
);

INSERT INTO sdplatform.sd_permission_api_meta (permission_id, api_uuid, resource, action)
SELECT sp.permission_id,
       COALESCE(p.id, uuid_generate_v4()),
       split_part(sp.name, ':', 1),
       split_part(sp.name, ':', 2)
FROM sdplatform.sd_permission sp
LEFT JOIN permissions p ON p.name = sp.name
WHERE sp.name IN ('users:read', 'users:create', 'roles:read', 'permissions:read', 'hr:read', 'hr:create')
ON CONFLICT (permission_id) DO UPDATE SET
  api_uuid = EXCLUDED.api_uuid,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;

INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, created_at)
SELECT r.role_id, p.permission_id, NOW()
FROM sdplatform.sd_role r
JOIN sdplatform.sd_permission p ON p.name IN ('users:read', 'users:create', 'roles:read', 'permissions:read', 'hr:read', 'hr:create')
WHERE r.name = 'admin'
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id AND rp.menu_id IS NULL
  );
