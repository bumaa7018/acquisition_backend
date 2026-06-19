CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Хэрэглэгч
CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name  VARCHAR(100) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_users_email      ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- Дүр
CREATE TABLE IF NOT EXISTS roles (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Эрх
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (resource, action)
);

-- Дүр ↔ Эрх
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Хэрэглэгч ↔ Дүр
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Нууц үг сэргээх токен
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_prt_token      ON password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_prt_expires_at ON password_reset_tokens (expires_at);

-- Газар зохион байгуулалтын төлөвлөгөө
CREATE TABLE IF NOT EXISTS plan (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_code    VARCHAR(100) NOT NULL UNIQUE,
    name         VARCHAR(255) NOT NULL,
    boundary_wkt TEXT         NOT NULL,
    area_m2      NUMERIC(18, 4),
    status       SMALLINT     NOT NULL DEFAULT 2,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
COMMENT ON COLUMN plan.status IS '1=DRAFT, 2=ACTIVE, 3=ARCHIVED';

-- Эх нэгж талбар (чөлөөлөлтын өмнөх бүртгэл)
CREATE TABLE IF NOT EXISTS source_parcel (
    id                 UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id          VARCHAR(12)  NOT NULL UNIQUE,
    plan_code          VARCHAR(100) NOT NULL REFERENCES plan(plan_code),
    geometry_wkt       TEXT         NOT NULL,
    au1_code           VARCHAR(10),
    au2_code           VARCHAR(10),
    au3_code           VARCHAR(10),
    right_type         VARCHAR(50),
    landuse            VARCHAR(100),
    area_m2            NUMERIC(18, 4),
    holder_last_name   VARCHAR(100),
    holder_name        VARCHAR(100),
    holder_register_no VARCHAR(20),
    holder_phone       VARCHAR(20),
    holder_email       VARCHAR(255),
    holder_type        VARCHAR(50),
    holder_civil_id    VARCHAR(50),
    app_no             VARCHAR(100),
    decision_no        VARCHAR(100),
    decision_date      DATE,
    contract_no        VARCHAR(100),
    contract_date      DATE,
    certificate_no     VARCHAR(100),
    certificate_date   DATE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_source_parcel_plan_code ON source_parcel (plan_code);
