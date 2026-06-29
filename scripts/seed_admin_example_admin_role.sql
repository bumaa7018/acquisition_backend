-- ============================================================
-- Seed: admin@example.com хэрэглэгчид admin role оноох
-- Ажиллуулах: CGO_ENABLED=0 go run ./cmd/seed-sql -db auth scripts/seed_admin_example_admin_role.sql
-- Idempotent: давтан ажиллуулахад role/permission/user_roles давхардахгүй.
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

INSERT INTO permissions (id, name, description, resource, action, created_at)
VALUES
  (uuid_generate_v4(), 'users:read', 'Read users', 'users', 'read', NOW()),
  (uuid_generate_v4(), 'users:create', 'Create users', 'users', 'create', NOW()),
  (uuid_generate_v4(), 'users:update', 'Update users', 'users', 'update', NOW()),
  (uuid_generate_v4(), 'users:delete', 'Delete users', 'users', 'delete', NOW()),
  (uuid_generate_v4(), 'roles:read', 'Read roles', 'roles', 'read', NOW()),
  (uuid_generate_v4(), 'roles:create', 'Create roles', 'roles', 'create', NOW()),
  (uuid_generate_v4(), 'roles:update', 'Update roles', 'roles', 'update', NOW()),
  (uuid_generate_v4(), 'roles:delete', 'Delete roles', 'roles', 'delete', NOW()),
  (uuid_generate_v4(), 'permissions:read', 'Read permissions', 'permissions', 'read', NOW()),
  (uuid_generate_v4(), 'permissions:create', 'Create permissions', 'permissions', 'create', NOW()),
  (uuid_generate_v4(), 'permissions:delete', 'Delete permissions', 'permissions', 'delete', NOW()),
  (uuid_generate_v4(), 'land:read', 'Read land acquisitions', 'land', 'read', NOW()),
  (uuid_generate_v4(), 'land:create', 'Create land acquisitions', 'land', 'create', NOW()),
  (uuid_generate_v4(), 'land:update', 'Update land acquisitions', 'land', 'update', NOW()),
  (uuid_generate_v4(), 'land:delete', 'Delete land acquisitions', 'land', 'delete', NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;

INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES (uuid_generate_v4(), 'admin', 'System administrator with full access', NOW(), NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  updated_at = NOW();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.name IN (
    'users:read',
    'users:create',
    'users:update',
    'users:delete',
    'roles:read',
    'roles:create',
    'roles:update',
    'roles:delete',
    'permissions:read',
    'permissions:create',
    'permissions:delete',
    'land:read',
    'land:create',
    'land:update',
    'land:delete'
  )
ON CONFLICT DO NOTHING;

DO $$
DECLARE
  target_user_id UUID;
  admin_role_id UUID;
BEGIN
  SELECT id INTO target_user_id
  FROM users
  WHERE email = 'admin@example.com'
    AND deleted_at IS NULL
  LIMIT 1;

  IF target_user_id IS NULL THEN
    RAISE EXCEPTION 'admin@example.com хэрэглэгч олдсонгүй';
  END IF;

  SELECT id INTO admin_role_id
  FROM roles
  WHERE name = 'admin';

  INSERT INTO user_roles (user_id, role_id)
  VALUES (target_user_id, admin_role_id)
  ON CONFLICT DO NOTHING;

  RAISE NOTICE 'admin@example.com хэрэглэгчид admin role оноогдлоо';
END $$;
