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

-- Seed data is intentionally not inserted here. The sample external valuation
-- users and their roles are created by startup seeder step
-- `000025_main_external_valuation_users`, after the sdplatform-native schema is
-- in place. These legacy public tables remain only so older deployments at
-- version 92 can be migrated by 000093.
