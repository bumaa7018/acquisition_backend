-- «Буулгалтын мэдээлэл» — буулгасан байгууламжийн төрөл. Хэрэглэгч оруулах
-- үедээ шинэ төрлийг нэмж болно (хадгалахад нэг гүйлгээнд үүснэ). Нэр
-- давхардахгүй (том/жижиг үсэг, захын зай ялгахгүй).
CREATE TABLE IF NOT EXISTS public.land_release_structure_type (
    id         serial PRIMARY KEY,
    name       text NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_land_release_structure_type_name
    ON public.land_release_structure_type (lower(btrim(name)));

-- Өмнө оруулсан газрууд төрөлгүй (NULL) үлдэнэ; шинэ оруулалтад заавал.
ALTER TABLE public.land_release_import
    ADD COLUMN IF NOT EXISTS structure_type_id integer
        REFERENCES public.land_release_structure_type(id) ON DELETE RESTRICT;
ALTER TABLE public.land_release
    ADD COLUMN IF NOT EXISTS structure_type_id integer
        REFERENCES public.land_release_structure_type(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_land_release_structure_type
    ON public.land_release (structure_type_id);
