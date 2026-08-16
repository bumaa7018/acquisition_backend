DELETE FROM sdplatform.sd_role_permission rp
USING sdplatform.sd_role r, sdplatform.sd_permission p
WHERE rp.role_id = r.role_id
  AND rp.permission_id = p.permission_id
  AND r.name = 'admin'
  AND p.name IN ('users:read', 'users:create', 'roles:read', 'permissions:read', 'hr:read', 'hr:create')
  AND rp.menu_id IS NULL;

DELETE FROM sdplatform.sd_permission p
WHERE p.name IN ('users:read', 'users:create', 'roles:read', 'permissions:read', 'hr:read', 'hr:create');
