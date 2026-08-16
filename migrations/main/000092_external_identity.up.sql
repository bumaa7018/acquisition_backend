CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   TEXT         NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name  VARCHAR(100) NOT NULL,
    position   TEXT         NOT NULL DEFAULT '',
    employee_id UUID,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    tokens_valid_from TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_main_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_main_users_deleted_at ON users (deleted_at);

CREATE TABLE IF NOT EXISTS roles (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (resource, action)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO permissions (id, name, description, resource, action, created_at)
VALUES
  (uuid_generate_v4(), 'compensation:read', 'Нөхөх олговор/үнэлгээ харах', 'compensation', 'read', NOW()),
  (uuid_generate_v4(), 'compensation:create', 'Нөхөх олговор/үнэлгээ бүртгэх', 'compensation', 'create', NOW()),
  (uuid_generate_v4(), 'compensation:update', 'Нөхөх олговор/үнэлгээ засах, төлөв шилжүүлэх', 'compensation', 'update', NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;

INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES (uuid_generate_v4(), 'professional_org', 'Мэргэжлийн байгууллага', NOW(), NOW())
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  updated_at = NOW();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('compensation:read', 'compensation:create', 'compensation:update')
WHERE r.name = 'professional_org'
ON CONFLICT DO NOTHING;

DO $$
DECLARE
  pwd_hash TEXT := '$2a$10$BEioEBBJWmCfIqp/PlBYR.NFz99nnLBMc4zIP486B1Rmo1ZyR8TVy';
BEGIN
  INSERT INTO users (id, username, email, password, first_name, last_name, position, is_active, created_at, updated_at)
  VALUES
    (uuid_generate_v4(), 'valuation_org_1', 'valuation.org1@example.mn', pwd_hash, 'Үнэлгээний', 'Байгууллага 1', 'Мэргэжлийн байгууллага', true, NOW(), NOW()),
    (uuid_generate_v4(), 'valuation_org_2', 'valuation.org2@example.mn', pwd_hash, 'Үнэлгээний', 'Байгууллага 2', 'Мэргэжлийн байгууллага', true, NOW(), NOW()),
    (uuid_generate_v4(), 'valuation_org_3', 'valuation.org3@example.mn', pwd_hash, 'Үнэлгээний', 'Байгууллага 3', 'Мэргэжлийн байгууллага', true, NOW(), NOW())
  ON CONFLICT (email) DO UPDATE SET
    username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    position = EXCLUDED.position,
    is_active = true,
    deleted_at = NULL,
    updated_at = NOW();

  INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id
  FROM users u
  JOIN roles r ON r.name = 'professional_org'
  WHERE u.email IN ('valuation.org1@example.mn', 'valuation.org2@example.mn', 'valuation.org3@example.mn')
  ON CONFLICT DO NOTHING;
END $$;

