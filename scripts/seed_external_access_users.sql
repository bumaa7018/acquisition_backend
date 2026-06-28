-- ============================================================
-- Seed: external access roles + sample users
-- Ажиллуулах: psql "$AUTH_DATABASE_URL" -f scripts/seed_external_access_users.sql
-- Нууц үг: Testpass123!
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

INSERT INTO permissions (id, name, description, resource, action, created_at)
VALUES
  (uuid_generate_v4(), 'land:read',   'Read land acquisitions',   'land', 'read',   NOW()),
  (uuid_generate_v4(), 'land:create', 'Create land acquisitions', 'land', 'create', NOW()),
  (uuid_generate_v4(), 'land:update', 'Update land acquisitions', 'land', 'update', NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;

INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES
  (uuid_generate_v4(), 'professional_org', 'Мэргэжлийн байгууллага', NOW(), NOW()),
  (uuid_generate_v4(), 'mika', 'МИКА', NOW(), NOW()),
  (uuid_generate_v4(), 'finance_specialist', 'Санхүүгийн мэргэжилтэн', NOW(), NOW()),
  (uuid_generate_v4(), 'senior_specialist', 'Ахлах мэргэжилтэн', NOW(), NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  updated_at = NOW();

-- External roles need land:read for lists/detail. Editing is still constrained by
-- role/resource checks in application code and assignedOrSenior middleware.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = 'land:read'
WHERE r.name IN ('professional_org', 'mika', 'finance_specialist')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('land:read', 'land:create', 'land:update')
WHERE r.name = 'senior_specialist'
ON CONFLICT DO NOTHING;

DO $$
DECLARE
  pwd_hash TEXT := '$2a$10$BEioEBBJWmCfIqp/PlBYR.NFz99nnLBMc4zIP486B1Rmo1ZyR8TVy';
BEGIN
  INSERT INTO users (id, username, email, password, first_name, last_name, position, is_active, created_at, updated_at)
  VALUES
    (uuid_generate_v4(), 'professional_primary', 'professional.primary@example.mn', pwd_hash, 'Үндсэн', 'Үнэлгээчин', 'Мэргэжлийн байгууллага', true, NOW(), NOW()),
    (uuid_generate_v4(), 'professional_independent', 'professional.independent@example.mn', pwd_hash, 'Хөндлөнгийн', 'Үнэлгээчин', 'Хөндлөнгийн мэргэжлийн байгууллага', true, NOW(), NOW()),
    (uuid_generate_v4(), 'mika_user', 'mika@example.mn', pwd_hash, 'МИКА', 'Мэргэжилтэн', 'МИКА', true, NOW(), NOW()),
    (uuid_generate_v4(), 'finance_specialist', 'finance.specialist@example.mn', pwd_hash, 'Санхүү', 'Мэргэжилтэн', 'Санхүүгийн мэргэжилтэн', true, NOW(), NOW()),
    (uuid_generate_v4(), 'senior_specialist_seed', 'senior.specialist@example.mn', pwd_hash, 'Ахлах', 'Мэргэжилтэн', 'Ахлах мэргэжилтэн', true, NOW(), NOW())
  ON CONFLICT (email) DO UPDATE SET
    username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    position = EXCLUDED.position,
    is_active = true,
    updated_at = NOW();

  INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u
  JOIN roles r ON r.name = 'professional_org'
  WHERE u.email IN ('professional.primary@example.mn', 'professional.independent@example.mn')
  ON CONFLICT DO NOTHING;

  INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u
  JOIN roles r ON r.name = 'mika'
  WHERE u.email = 'mika@example.mn'
  ON CONFLICT DO NOTHING;

  INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u
  JOIN roles r ON r.name = 'finance_specialist'
  WHERE u.email = 'finance.specialist@example.mn'
  ON CONFLICT DO NOTHING;

  INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u
  JOIN roles r ON r.name = 'senior_specialist'
  WHERE u.email = 'senior.specialist@example.mn'
  ON CONFLICT DO NOTHING;
END $$;
