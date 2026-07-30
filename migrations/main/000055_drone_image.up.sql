-- Дроны ортофото (.tif) зургууд — газар чөлөөлөлтөд холбогдоно.
-- Файл нь GeoServer уншиж чадах хуваалцсан хавтсанд хадгалагдаж, GeoServer-ийн
-- ImageMosaic давхаргад granule болж бүртгэгдэнэ. file_name нь mosaic-ийн
-- `location` атрибуттай тэнцүү — WMS дуудахад CQL_FILTER-ээр тэрийг явуулна.
CREATE TABLE IF NOT EXISTS drone_image (
    id             UUID PRIMARY KEY,
    acquisition_id UUID NOT NULL REFERENCES land_acquisition(id) ON DELETE CASCADE,
    file_name      TEXT NOT NULL UNIQUE,
    original_name  TEXT NOT NULL,
    content_type   TEXT NOT NULL DEFAULT '',
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    -- GeoServer-ээс granule-ийн хүрээг уншиж дүүргэнэ (WGS84). Уншиж чадаагүй
    -- үед NULL — давхарга харуулахад шаардлагагүй, зөвхөн "зураг дээр очих"-д.
    min_x          DOUBLE PRECISION,
    min_y          DOUBLE PRECISION,
    max_x          DOUBLE PRECISION,
    max_y          DOUBLE PRECISION,
    -- GeoServer-ийн mosaic-д granule болж бүртгэгдсэн эсэх
    published      BOOLEAN NOT NULL DEFAULT FALSE,
    uploaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by    UUID
);

CREATE INDEX IF NOT EXISTS idx_drone_image_acquisition
    ON drone_image (acquisition_id, uploaded_at DESC);
