-- ХУУЧИН МЭДЭЭЛЛИЙН ИМПОРТЫН ТҮР ХҮСНЭГТҮҮД (internal/legacyimport).
--
-- Админ чөлөөлөлтийн хил, нэгж талбарын shapefile, үнэлгээний эксэлийг
-- оруулахад задалсан мөрүүд ЭНД хадгалагдана. Системийн хүснэгтүүдэд
-- (land_acquisition, parcel, compensation …) «Оруулах» дарагдах хүртэл НЭГ Ч
-- мөр бичигдэхгүй. Геометр нь PostGIS баганад (EPSG:4326) тул «хилийн гадна»,
-- «төлөвлөгөөний хилтэй таарах» шалгалтыг санд нь бодно.
--
-- «Оруулах» амжилттай болмогц boundary/parcel/report_row мөрүүд устана;
-- session-д шалгалтын үр дүн ба «Оруулах»-ын тэмдэглэл түүх болж үлдэнэ.
CREATE SCHEMA IF NOT EXISTS legacy_import_stage;

CREATE TABLE IF NOT EXISTS legacy_import_stage.session (
    id uuid PRIMARY KEY,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- draft → committing → committed | failed
    status text NOT NULL DEFAULT 'draft',
    category jsonb,
    boundary jsonb,
    parcels jsonb,
    reports jsonb,
    result jsonb,
    CONSTRAINT legacy_import_session_status CHECK (status IN ('draft', 'committing', 'committed', 'failed'))
);

CREATE TABLE IF NOT EXISTS legacy_import_stage.boundary (
    session_id uuid NOT NULL REFERENCES legacy_import_stage.session(id) ON DELETE CASCADE,
    idx int NOT NULL,
    file_name text NOT NULL,
    plan_parcel_id text NOT NULL DEFAULT '',
    row jsonb NOT NULL,
    -- ГУС /plan/project-ийн хариу (олдоогүй бол NULL)
    plan jsonb,
    geom geometry(Geometry, 4326),
    issues jsonb NOT NULL DEFAULT '[]',
    PRIMARY KEY (session_id, idx)
);
CREATE INDEX IF NOT EXISTS legacy_import_boundary_geom ON legacy_import_stage.boundary USING gist (geom);

CREATE TABLE IF NOT EXISTS legacy_import_stage.parcel (
    session_id uuid NOT NULL REFERENCES legacy_import_stage.session(id) ON DELETE CASCADE,
    idx int NOT NULL,
    file_name text NOT NULL,
    -- Харуулах дугаар (дугааргүй бол ERR… зохиомол)
    parcel_id text NOT NULL DEFAULT '',
    row jsonb NOT NULL,
    geom geometry(Geometry, 4326),
    area_m2 numeric,
    boundary_idx int,
    overlap_ratio numeric,
    outside boolean NOT NULL DEFAULT false,
    nearest_distance_m numeric,
    -- info | ub | missing | failed | skip
    gus_origin text,
    -- ГУС-ын хариу — «Оруулах» үед ДАХИН дуудахгүй
    source jsonb,
    issues jsonb NOT NULL DEFAULT '[]',
    PRIMARY KEY (session_id, idx)
);
CREATE INDEX IF NOT EXISTS legacy_import_parcel_geom ON legacy_import_stage.parcel USING gist (geom);
CREATE INDEX IF NOT EXISTS legacy_import_parcel_id ON legacy_import_stage.parcel (session_id, parcel_id);

CREATE TABLE IF NOT EXISTS legacy_import_stage.report_row (
    session_id uuid NOT NULL REFERENCES legacy_import_stage.session(id) ON DELETE CASCADE,
    idx int NOT NULL,
    file_name text NOT NULL,
    parcel_id text NOT NULL DEFAULT '',
    row jsonb NOT NULL,
    -- staged | db | gus:info | gus:ub | missing
    match text,
    source jsonb,
    issues jsonb NOT NULL DEFAULT '[]',
    PRIMARY KEY (session_id, idx)
);

COMMENT ON SCHEMA legacy_import_stage IS
    'Хуучин мэдээллийн импортын түр хүснэгтүүд — «Оруулах» хүртэл системийн хүснэгтэд бичихгүй.';
