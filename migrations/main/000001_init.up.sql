CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- 1. газар чөлөөлөлт
CREATE TABLE IF NOT EXISTS land_acquisition (
    id               UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id          UUID,
    plan_code        VARCHAR(100)   NOT NULL,
    plan_name        TEXT,
    plan_geom        geometry(Polygon, 4326),
    plan_area_m2     NUMERIC(18, 4),
    geometry         geometry(Polygon, 4326),
    area_m2          NUMERIC(18, 4),
    status           SMALLINT       NOT NULL DEFAULT 1,
    start_date       DATE,
    end_date         DATE,
    acquisition_name TEXT,
    implementing_org TEXT,
    reason           TEXT,
    responsible_org  TEXT,
    funding_source   TEXT,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_by       VARCHAR(255),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_by       VARCHAR(255)
);
COMMENT ON COLUMN land_acquisition.status IS '1=NEW, 2=FIELD, 3=VERIFIED, 4=CANCELLED';
COMMENT ON COLUMN land_acquisition.geometry IS 'PostGIS geometry(Polygon, 4326)';
COMMENT ON COLUMN land_acquisition.plan_geom IS 'PostGIS geometry(Polygon, 4326)';

CREATE INDEX IF NOT EXISTS idx_land_acquisition_status    ON land_acquisition (status);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_code ON land_acquisition (plan_code);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_geom ON land_acquisition USING GIST (plan_geom);
CREATE INDEX IF NOT EXISTS idx_land_acquisition_geometry  ON land_acquisition USING GIST (geometry);

-- 2. аймаг (au1)
CREATE TABLE IF NOT EXISTS au1 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    area_m2    NUMERIC(18, 4),
    geometry   geometry(Polygon, 4326),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_au1_geometry ON au1 USING GIST (geometry);

-- 3. сум (au2)
CREATE TABLE IF NOT EXISTS au2 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    au1_code   VARCHAR(10)  NOT NULL REFERENCES au1(code),
    area_m2    NUMERIC(18, 4),
    geometry   geometry(Polygon, 4326),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_au2_au1_code ON au2 (au1_code);
CREATE INDEX IF NOT EXISTS idx_au2_geometry ON au2 USING GIST (geometry);

-- 4. баг (au3)
CREATE TABLE IF NOT EXISTS au3 (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    au1_code   VARCHAR(10)  NOT NULL REFERENCES au1(code),
    au2_code   VARCHAR(10)  NOT NULL REFERENCES au2(code),
    area_m2    NUMERIC(18, 4),
    geometry   geometry(Polygon, 4326),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255)
);
CREATE INDEX IF NOT EXISTS idx_au3_au1_code ON au3 (au1_code);
CREATE INDEX IF NOT EXISTS idx_au3_au2_code ON au3 (au2_code);
CREATE INDEX IF NOT EXISTS idx_au3_geometry ON au3 USING GIST (geometry);

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
    geometry            geometry(Polygon, 4326),
    acquisition_geom    geometry(Polygon, 4326),
    acquisition_area_m2 NUMERIC(18, 4),
    right_type          INTEGER,
    landuse             VARCHAR(100),
    area_m2             NUMERIC(18, 4),
    compensation_paid   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255)
);
COMMENT ON COLUMN parcel.geometry         IS 'PostGIS geometry(Polygon, 4326)';
COMMENT ON COLUMN parcel.acquisition_geom IS 'Нэгж талбар болон чөлөөлөлтийн бүсийн давхцах хэсгийн PostGIS geometry(Polygon, 4326)';
COMMENT ON COLUMN parcel.acquisition_area_m2 IS 'Давхцах хэсгийн талбай (м²)';

CREATE INDEX IF NOT EXISTS idx_parcel_acquisition_id   ON parcel (acquisition_id);
CREATE INDEX IF NOT EXISTS idx_parcel_au_codes         ON parcel (au1_code, au2_code, au3_code);
CREATE INDEX IF NOT EXISTS idx_parcel_geometry         ON parcel USING GIST (geometry);
CREATE INDEX IF NOT EXISTS idx_parcel_acquisition_geom ON parcel USING GIST (acquisition_geom);

-- 8. нэгж талбарын эрх зүйн дэлгэрэнгүй мэдээлэл
CREATE TABLE IF NOT EXISTS parcel_detail (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id        UUID        REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id   UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    right_type       INTEGER,
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

-- 9. газар чөлөөлөлтийн явцын түүх
CREATE TABLE IF NOT EXISTS acquisition_progress (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    acquisition_id UUID        NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    from_status    INT         NOT NULL,
    to_status      INT         NOT NULL,
    note           TEXT,
    changed_by     TEXT        NOT NULL DEFAULT 'system',
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_acq_progress_acq_id ON acquisition_progress (acquisition_id);

-- 10. нэгж талбарын төлбөр
CREATE TABLE IF NOT EXISTS parcel_payment (
    id         UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id  UUID          NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    amount     NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency   VARCHAR(10)   NOT NULL DEFAULT 'MNT',
    paid_at    DATE,
    note       TEXT,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100)
);
CREATE INDEX IF NOT EXISTS idx_parcel_payment_parcel ON parcel_payment (parcel_id);

-- 11. баримт бичгийн мета мэдээлэл
CREATE TABLE IF NOT EXISTS document (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT        NOT NULL,
    file_type   TEXT        NOT NULL,
    file_url    TEXT        NOT NULL,
    size_bytes  BIGINT      NOT NULL DEFAULT 0,
    note        TEXT,
    uploaded_by TEXT        NOT NULL DEFAULT 'system',
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 12. газар чөлөөлөлт ↔ баримт бичиг
CREATE TABLE IF NOT EXISTS acquisition_document (
    acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    document_id    UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    PRIMARY KEY (acquisition_id, document_id)
);
CREATE INDEX IF NOT EXISTS idx_acq_doc_acquisition ON acquisition_document (acquisition_id);

-- 13. нэгж талбар ↔ баримт бичиг
CREATE TABLE IF NOT EXISTS parcel_document (
    parcel_id   UUID NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    PRIMARY KEY (parcel_id, document_id)
);
CREATE INDEX IF NOT EXISTS idx_parcel_doc_parcel ON parcel_document (parcel_id);
