-- «Буулгалтын мэдээлэл» — газрын зураг. Оруулсан зургуудыг (frontend) НЭГ PDF
-- болгож хадгална (MinIO, namespace land-release). Олноор оруулахад нэг PDF
-- тухайн оруулалтын БҮХ газарт холбоос хүснэгтээр холбогдоно; дэлгэрэнгүйгээс
-- оруулсан нь зөвхөн тэр газарт.
CREATE TABLE IF NOT EXISTS public.land_release_photo (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name  text NOT NULL,
    file_url   text NOT NULL,
    size_bytes bigint NOT NULL DEFAULT 0,
    page_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS public.land_release_photo_link (
    release_id uuid NOT NULL REFERENCES public.land_release(id) ON DELETE CASCADE,
    photo_id   uuid NOT NULL REFERENCES public.land_release_photo(id) ON DELETE CASCADE,
    linked_at  timestamptz NOT NULL DEFAULT now(),
    linked_by  text NOT NULL DEFAULT '',
    PRIMARY KEY (release_id, photo_id)
);
CREATE INDEX IF NOT EXISTS idx_land_release_photo_link_photo
    ON public.land_release_photo_link (photo_id);
