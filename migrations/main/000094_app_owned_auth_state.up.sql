-- Аппын өөрийн нэвтрэлтийн туслах төлөв.
--
-- Эдгээр нь sdplatform схемд БАЙХГҮЙ ойлголтууд (нууц үг сэргээх токен,
-- хүчингүй болсон JWT, сессийн эх цэг) тул тэр схемд хүснэгт нэмэхийн оронд
-- энэ буюу аппын ӨӨРИЙН өгөгдлийн санд хадгална. Ингэснээр sdplatform DB рүү
-- залгахад бид тэнд юу ч бичихгүй.
--
-- user_source: хэрэглэгчийн эх сурвалж — 'auth' (sdplatform DB) эсвэл 'main'
-- (энэ DB дэх гадаад байгууллагууд). Хоёр эх сурвалж тус тусын serial4
-- дараалалтай тул ID нь мөргөлдөж болзошгүй; түлхүүрт эх сурвалж орсноор
-- өөр хүний сесс/токен хольцолдохоос сэргийлнэ.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id          uuid         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_source varchar(16)  NOT NULL DEFAULT 'auth',
    user_id     int4         NOT NULL,
    token       varchar(255) NOT NULL UNIQUE,
    expires_at  timestamptz  NOT NULL,
    used        boolean      NOT NULL DEFAULT false,
    created_at  timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_prt_token ON password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_prt_expires_at ON password_reset_tokens (expires_at);

CREATE TABLE IF NOT EXISTS revoked_tokens (
    jti         varchar(255) PRIMARY KEY,
    user_source varchar(16)  NOT NULL DEFAULT 'auth',
    user_id     int4         NOT NULL,
    token_type  varchar(32)  NOT NULL,
    expires_at  timestamptz  NOT NULL,
    revoked_at  timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user ON revoked_tokens (user_source, user_id);

CREATE TABLE IF NOT EXISTS user_token_epoch (
    user_source varchar(16) NOT NULL DEFAULT 'auth',
    user_id     int4        NOT NULL,
    valid_from  timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_source, user_id)
);
