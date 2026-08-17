-- Газар зохион байгуулалтын төлөвлөгөө (main DB).
--
-- Эх сурвалж нь дундын сервисийн (middleware) /plan/project — чөлөөлөлт үүсгэх /
-- хил шинэчлэх үед татагдаж ЭНД хадгалагдана. Төлөвлөгөөгөөр хайх, сонгох
-- үйлдлүүд (жишээ нь /plans/suggest) auth DB-ийн төсөөллийн plan хүснэгтийн
-- оронд үүнээс уншина.
--
-- boundary_wkt нь NULL байж болно: /plan/project хил буцаадаггүй бөгөөд хил нь
-- зөвхөн хуучин (auth DB-ээс татагдсан) төлөвлөгөөнүүдэд бий.
CREATE TABLE IF NOT EXISTS plan (
    id             UUID          PRIMARY KEY,
    plan_code      VARCHAR(100)  NOT NULL UNIQUE,
    project_id     VARCHAR(100),
    name           TEXT          NOT NULL DEFAULT '',
    plan_type_name TEXT,
    boundary_wkt   public.geometry(Geometry, 4326),
    area_m2        NUMERIC(18, 4),
    status         SMALLINT      NOT NULL DEFAULT 0,
    start_data     TEXT,
    end_data       TEXT,
    approved_date  TEXT,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE plan IS 'Газар зохион байгуулалтын төлөвлөгөө — middleware /plan/project-ээс татагдаж хадгалагдана';
COMMENT ON COLUMN plan.project_id IS 'Дундын сервис дэх төслийн ID';
COMMENT ON COLUMN plan.boundary_wkt IS 'PostGIS geometry(Geometry, 4326). /plan/project хил буцаахгүй тул NULL байж болно';

CREATE INDEX IF NOT EXISTS idx_plan_name ON plan (name);
CREATE INDEX IF NOT EXISTS idx_plan_boundary_wkt ON plan USING GIST (boundary_wkt);
