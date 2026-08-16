CREATE TABLE IF NOT EXISTS public.users (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   TEXT         NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name  VARCHAR(100) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    position   TEXT         NOT NULL DEFAULT '',
    tokens_valid_from TIMESTAMPTZ,
    employee_id UUID NULL REFERENCES public.employee(id)
);
CREATE INDEX IF NOT EXISTS idx_users_email
    ON public.users (email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON public.users (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS ux_users_employee_active
    ON public.users(employee_id) WHERE employee_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS public.roles (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.permissions (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (resource, action)
);

CREATE TABLE IF NOT EXISTS public.role_permissions (
    role_id       UUID NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES public.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS public.user_roles (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_prt_token
    ON public.password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_prt_expires_at
    ON public.password_reset_tokens (expires_at);

CREATE TABLE IF NOT EXISTS public.revoked_tokens (
    jti        VARCHAR(255) PRIMARY KEY,
    user_id    UUID        NOT NULL,
    token_type VARCHAR(32) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at
    ON public.revoked_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id
    ON public.revoked_tokens (user_id);

CREATE TABLE IF NOT EXISTS public.systems (
    system_id   INT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS public.system_roles (
    system_role_id BIGSERIAL PRIMARY KEY,
    system_id      INT  NOT NULL REFERENCES public.systems(system_id) ON DELETE CASCADE,
    role_id        UUID NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ,
    UNIQUE (system_id, role_id)
);

CREATE TABLE IF NOT EXISTS public.menus (
    menu_id    BIGSERIAL PRIMARY KEY,
    code       VARCHAR(32) UNIQUE,
    name       TEXT,
    parent_id  BIGINT REFERENCES public.menus(menu_id) ON DELETE SET NULL,
    menu_url   TEXT,
    sort_order INT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID,
    menu_icon  TEXT,
    parent_code VARCHAR(32),
    system_id  INT REFERENCES public.systems(system_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS public.menu_roles (
    menu_id    BIGINT NOT NULL REFERENCES public.menus(menu_id) ON DELETE CASCADE,
    role_id    UUID   NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (menu_id, role_id)
);

CREATE TABLE IF NOT EXISTS public.forms (
    form_id    BIGSERIAL PRIMARY KEY,
    form_name  TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    code       VARCHAR(32) UNIQUE,
    system_id  INT REFERENCES public.systems(system_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS public.menu_forms (
    menu_form_id BIGSERIAL PRIMARY KEY,
    form_id      BIGINT REFERENCES public.forms(form_id) ON DELETE CASCADE,
    menu_id      BIGINT REFERENCES public.menus(menu_id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ,
    form_code    VARCHAR(32),
    menu_code    VARCHAR(32),
    UNIQUE (form_id, menu_id),
    UNIQUE (form_code, menu_code)
);

CREATE TABLE IF NOT EXISTS public.form_permissions (
    form_permission_id BIGSERIAL PRIMARY KEY,
    form_id       BIGINT REFERENCES public.forms(form_id) ON DELETE CASCADE,
    permission_id UUID REFERENCES public.permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    form_code     VARCHAR(32),
    permission_code VARCHAR(100),
    UNIQUE (form_id, permission_id),
    UNIQUE (form_code, permission_code)
);

INSERT INTO public.users (
    id, username, email, password, first_name, last_name, is_active,
    created_at, updated_at, deleted_at, position, tokens_valid_from, employee_id
)
SELECT m.api_uuid,
       COALESCE(u.username, ''),
       COALESCE(u.email, ''),
       COALESCE(u.password, ''),
       COALESCE(u.firstname, ''),
       COALESCE(u.lastname, ''),
       COALESCE(u.is_active, false),
       COALESCE(u.created_at::timestamptz, NOW()),
       COALESCE(u.updated_at::timestamptz, NOW()),
       u.deleted_at::timestamptz,
       COALESCE(m.position, ''),
       m.tokens_valid_from,
       m.employee_uuid
FROM sdplatform.sd_user u
JOIN sdplatform.sd_user_api_meta m ON m.user_id = u.user_id
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.password_reset_tokens (id, user_id, token, expires_at, used, created_at)
SELECT prt.id, prt.user_id, prt.token, prt.expires_at, prt.used, prt.created_at
FROM sdplatform.password_reset_tokens prt
WHERE EXISTS (SELECT 1 FROM public.users u WHERE u.id = prt.user_id)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.revoked_tokens (jti, user_id, token_type, expires_at, revoked_at)
SELECT jti, user_id, token_type, expires_at, revoked_at
FROM sdplatform.revoked_tokens
ON CONFLICT (jti) DO NOTHING;
