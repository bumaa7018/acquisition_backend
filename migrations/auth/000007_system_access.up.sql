CREATE TABLE IF NOT EXISTS systems (
    system_id   INT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS system_roles (
    system_role_id BIGSERIAL PRIMARY KEY,
    system_id      INT  NOT NULL REFERENCES systems(system_id) ON DELETE CASCADE,
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ,
    UNIQUE (system_id, role_id)
);

CREATE TABLE IF NOT EXISTS menus (
    menu_id    BIGSERIAL PRIMARY KEY,
    code       VARCHAR(32) UNIQUE,
    name       TEXT,
    parent_id  BIGINT REFERENCES menus(menu_id) ON DELETE SET NULL,
    menu_url   TEXT,
    sort_order INT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID,
    menu_icon  TEXT,
    parent_code VARCHAR(32),
    system_id  INT REFERENCES systems(system_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS menu_roles (
    menu_id    BIGINT NOT NULL REFERENCES menus(menu_id) ON DELETE CASCADE,
    role_id    UUID   NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (menu_id, role_id)
);

CREATE TABLE IF NOT EXISTS forms (
    form_id    BIGSERIAL PRIMARY KEY,
    form_name  TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    code       VARCHAR(32) UNIQUE,
    system_id  INT REFERENCES systems(system_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS menu_forms (
    menu_form_id BIGSERIAL PRIMARY KEY,
    form_id      BIGINT REFERENCES forms(form_id) ON DELETE CASCADE,
    menu_id      BIGINT REFERENCES menus(menu_id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ,
    form_code    VARCHAR(32),
    menu_code    VARCHAR(32),
    UNIQUE (form_id, menu_id),
    UNIQUE (form_code, menu_code)
);

CREATE TABLE IF NOT EXISTS form_permissions (
    form_permission_id BIGSERIAL PRIMARY KEY,
    form_id       BIGINT REFERENCES forms(form_id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    form_code     VARCHAR(32),
    permission_code VARCHAR(100),
    UNIQUE (form_id, permission_id),
    UNIQUE (form_code, permission_code)
);

INSERT INTO systems (system_id, name, description)
VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
ON CONFLICT (system_id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

-- revoked_tokens нь authDB болон mainDB хэрэглэгчийн JTI-г хоёуланг нь хадгална.
-- Гадны хэрэглэгч mainDB-д байрших тул authDB.users FK-г сулруулна.
ALTER TABLE revoked_tokens DROP CONSTRAINT IF EXISTS revoked_tokens_user_id_fkey;

