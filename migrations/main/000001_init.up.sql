CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- 1. газар чөлөөлөлт
CREATE TABLE IF NOT EXISTS land_acquisition (
    id           UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id      UUID,
    plan_code    VARCHAR(100)   NOT NULL,
    plan_geom    TEXT,
    plan_area_m2 NUMERIC(18, 4),
    geometry     TEXT,
    area_m2      NUMERIC(18, 4),
    status       SMALLINT       NOT NULL DEFAULT 2,
    start_date   DATE,
    end_date     DATE,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_by   VARCHAR(255),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_by   VARCHAR(255)
);
COMMENT ON COLUMN land_acquisition.status IS '1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED';
COMMENT ON COLUMN land_acquisition.geometry IS 'WKT хэлбэрт (POLYGON(...))';
COMMENT ON COLUMN land_acquisition.plan_geom IS 'WKT хэлбэрт (POLYGON(...))';

CREATE INDEX IF NOT EXISTS idx_land_acquisition_status    ON land_acquisition (status);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_code ON land_acquisition (plan_code);

-- 2. аймаг (au1)
CREATE TABLE IF NOT EXISTS au1 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    area_m2    NUMERIC(18, 4),
    geometry   TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
COMMENT ON COLUMN au1.geometry IS 'WKT хэлбэрт';

-- 3. сум (au2)
CREATE TABLE IF NOT EXISTS au2 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    au1_code   VARCHAR(10)  NOT NULL REFERENCES au1(code),
    area_m2    NUMERIC(18, 4),
    geometry   TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_au2_au1_code ON au2 (au1_code);

-- 4. баг (au3)
CREATE TABLE IF NOT EXISTS au3 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    au1_code   VARCHAR(10)  NOT NULL REFERENCES au1(code),
    au2_code   VARCHAR(10)  NOT NULL REFERENCES au2(code),
    area_m2    NUMERIC(18, 4),
    geometry   TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_au3_au1_code ON au3 (au1_code);
CREATE INDEX IF NOT EXISTS idx_au3_au2_code ON au3 (au2_code);

-- 5. газар чөлөөлөлт ↔ захиргааны нэгж холбоос
CREATE TABLE IF NOT EXISTS land_acquisition_au (
    land_acquisition_id UUID         NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    au1_code            VARCHAR(10),
    au1_name            VARCHAR(100),
    au2_code            VARCHAR(10),
    au2_name            VARCHAR(100),
    au3_code            VARCHAR(10)  NOT NULL,
    au3_name            VARCHAR(100),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255),
    PRIMARY KEY (land_acquisition_id, au3_code)
);

-- 6. газар эзэмшигч / өмчлөгч
CREATE TABLE IF NOT EXISTS right_holder (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    last_name   VARCHAR(100),
    name        VARCHAR(100) NOT NULL,
    register_no VARCHAR(20)  UNIQUE,
    address     TEXT,
    phone       VARCHAR(20),
    email       VARCHAR(255),
    holder_type VARCHAR(50),
    civil_id    VARCHAR(50),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(255),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by  VARCHAR(255)
);

-- 7. нэгж талбар
CREATE TABLE IF NOT EXISTS parcel (
    id                  UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id           VARCHAR(12)  NOT NULL UNIQUE,
    old_parcel_id       VARCHAR(14),
    acquisition_id      UUID         NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    valid_from          DATE,
    valid_till          DATE,
    au1_code            VARCHAR(10),
    au2_code            VARCHAR(10),
    au3_code            VARCHAR(10),
    geometry            TEXT,
    acquisition_geom    TEXT,
    acquisition_area_m2 NUMERIC(18, 4),
    right_type          VARCHAR(50),
    landuse             VARCHAR(100),
    area_m2             NUMERIC(18, 4),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255)
);
COMMENT ON COLUMN parcel.geometry         IS 'WKT хэлбэрт';
COMMENT ON COLUMN parcel.acquisition_geom IS 'Нэгж талбар болон чөлөөлөлтийн бүсийн давхцах хэсгийн геометр (WKT)';
COMMENT ON COLUMN parcel.acquisition_area_m2 IS 'Давхцах хэсгийн талбай (м²)';

CREATE INDEX IF NOT EXISTS idx_parcel_acquisition_id ON parcel (acquisition_id);
CREATE INDEX IF NOT EXISTS idx_parcel_au_codes       ON parcel (au1_code, au2_code, au3_code);

-- 8. нэгж талбарын эрх зүйн дэлгэрэнгүй мэдээлэл
CREATE TABLE IF NOT EXISTS parcel_detail (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id        UUID        REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id   UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    right_type       VARCHAR(50),
    holder_id        UUID        REFERENCES right_holder(id),
    app_no           VARCHAR(100),
    app_timestamps   TIMESTAMPTZ,
    decision_no      VARCHAR(100),
    decision_date    DATE,
    contract_no      VARCHAR(100),
    contract_date    DATE,
    certificate_no   VARCHAR(100),
    certificate_date DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       VARCHAR(255),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by       VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_parcel_detail_parcel_id      ON parcel_detail (parcel_id);
CREATE INDEX IF NOT EXISTS idx_parcel_detail_acquisition_id ON parcel_detail (acquisition_id);
CREATE INDEX IF NOT EXISTS idx_parcel_detail_holder_id      ON parcel_detail (holder_id);
