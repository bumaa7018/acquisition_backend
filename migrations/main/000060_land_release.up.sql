-- «Буулгалтын мэдээлэл» — shapefile-аар оруулсан газрууд (буулгах/чөлөөлөх
-- талбай). Үндсэн нэгж талбар (parcel) / чөлөөлөлттэй ХОЛИЛДОХГҮЙ тусдаа
-- хүснэгтүүд.
--
-- Төлөв (land_release.status):
--   1 — Төлөвлөж байгаа
--   2 — Захирамж гарч чөлөөлсөн (захирамж заавал)
--   3 — Хүчингүй
-- Шилжилт: 1 → 2, 1 → 3.

-- Захирамж — НЭГ дугаар, НЭГ файл. Олноор оруулахад нэг захирамж олон газарт
-- холбоос хүснэгтээр холбогдоно; нэг газрын төлөв солиход шинэ мөр үүснэ.
CREATE TABLE IF NOT EXISTS public.land_release_decree (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    decree_number varchar(100) NOT NULL,
    decree_date   date,
    file_name     text NOT NULL,
    file_type     text NOT NULL DEFAULT '',
    file_url      text NOT NULL,
    size_bytes    bigint NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_land_release_decree_number
    ON public.land_release_decree (decree_number);

-- Нэг оруулалт (popup-аар нэг удаа оруулсан файл(ууд)).
CREATE TABLE IF NOT EXISTS public.land_release_import (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text NOT NULL,
    description    text NOT NULL DEFAULT '',
    initial_status smallint NOT NULL,
    file_names     text[] NOT NULL DEFAULT '{}',
    feature_count  integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL DEFAULT ''
);

-- Газар (shapefile-ийн нэг бичлэг). release_no — системийн дугаар.
CREATE TABLE IF NOT EXISTS public.land_release (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    release_no   bigint GENERATED ALWAYS AS IDENTITY,
    import_id    uuid REFERENCES public.land_release_import(id) ON DELETE SET NULL,
    name         text NOT NULL,
    description  text NOT NULL DEFAULT '',
    status       smallint NOT NULL DEFAULT 1 CHECK (status IN (1, 2, 3)),
    geometry     geometry(MultiPolygon, 4326) NOT NULL,
    area_m2      numeric(18, 2) NOT NULL DEFAULT 0,
    source_file  text NOT NULL DEFAULT '',
    source_index integer NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL DEFAULT '',
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text NOT NULL DEFAULT '',
    CONSTRAINT uq_land_release_no UNIQUE (release_no)
);
CREATE INDEX IF NOT EXISTS idx_land_release_geometry ON public.land_release USING gist (geometry);
CREATE INDEX IF NOT EXISTS idx_land_release_status ON public.land_release (status);
CREATE INDEX IF NOT EXISTS idx_land_release_import ON public.land_release (import_id);

-- Газар ↔ захирамж. Газар бүр ЗӨВХӨН нэг захирамжтай (PK = release_id).
CREATE TABLE IF NOT EXISTS public.land_release_decree_link (
    release_id uuid PRIMARY KEY REFERENCES public.land_release(id) ON DELETE CASCADE,
    decree_id  uuid NOT NULL REFERENCES public.land_release_decree(id) ON DELETE RESTRICT,
    linked_at  timestamptz NOT NULL DEFAULT now(),
    linked_by  text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_land_release_decree_link_decree
    ON public.land_release_decree_link (decree_id);

-- Төлөвийн түүх (оруулалт ч энд эхний мөр болж бичигдэнэ: from_status NULL).
CREATE TABLE IF NOT EXISTS public.land_release_status_history (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id  uuid NOT NULL REFERENCES public.land_release(id) ON DELETE CASCADE,
    from_status smallint,
    to_status   smallint NOT NULL,
    decree_id   uuid REFERENCES public.land_release_decree(id) ON DELETE SET NULL,
    note        text NOT NULL DEFAULT '',
    changed_at  timestamptz NOT NULL DEFAULT now(),
    changed_by  text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_land_release_status_history_release
    ON public.land_release_status_history (release_id, changed_at DESC);
