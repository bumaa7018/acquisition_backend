CREATE TABLE IF NOT EXISTS sdplatform.password_reset_tokens (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES sdplatform.sd_user_api_meta(api_uuid) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sdplatform_prt_token
    ON sdplatform.password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_sdplatform_prt_expires_at
    ON sdplatform.password_reset_tokens (expires_at);

CREATE TABLE IF NOT EXISTS sdplatform.revoked_tokens (
    jti        VARCHAR(255) PRIMARY KEY,
    user_id    UUID         NOT NULL,
    token_type VARCHAR(32)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sdplatform_revoked_tokens_expires_at
    ON sdplatform.revoked_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_sdplatform_revoked_tokens_user_id
    ON sdplatform.revoked_tokens (user_id);

DO $$
BEGIN
    IF to_regclass('public.password_reset_tokens') IS NOT NULL THEN
        INSERT INTO sdplatform.password_reset_tokens (id, user_id, token, expires_at, used, created_at)
        SELECT id, user_id, token, expires_at, used, created_at
        FROM public.password_reset_tokens prt
        WHERE EXISTS (
            SELECT 1
            FROM sdplatform.sd_user_api_meta m
            WHERE m.api_uuid = prt.user_id
        )
        ON CONFLICT (id) DO NOTHING;
    END IF;

    IF to_regclass('public.revoked_tokens') IS NOT NULL THEN
        INSERT INTO sdplatform.revoked_tokens (jti, user_id, token_type, expires_at, revoked_at)
        SELECT jti, user_id, token_type, expires_at, revoked_at
        FROM public.revoked_tokens
        ON CONFLICT (jti) DO NOTHING;
    END IF;
END $$;

DROP TABLE IF EXISTS public.form_permissions;
DROP TABLE IF EXISTS public.menu_forms;
DROP TABLE IF EXISTS public.menu_roles;
DROP TABLE IF EXISTS public.menus;
DROP TABLE IF EXISTS public.forms;
DROP TABLE IF EXISTS public.system_roles;
DROP TABLE IF EXISTS public.systems;
DROP TABLE IF EXISTS public.password_reset_tokens;
DROP TABLE IF EXISTS public.revoked_tokens;
DROP TABLE IF EXISTS public.user_roles;
DROP TABLE IF EXISTS public.role_permissions;
DROP TABLE IF EXISTS public.permissions;
DROP TABLE IF EXISTS public.roles;
DROP TABLE IF EXISTS public.users;
