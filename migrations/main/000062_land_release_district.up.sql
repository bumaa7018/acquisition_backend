-- «Буулгалтын мэдээлэл» — газрын сум/дүүрэг ба аймаг/хот (au2/au1). Оруулах
-- үед байршлаар тодорхойлж хадгална (landreleasesql.AssignDistrictsSQL).
-- Өмнө оруулсан газруудыг seeder алхам 000062_land_release_district нөхнө.
-- NULL — ямар ч сум/дүүрэгтэй давхцаагүй.
ALTER TABLE public.land_release
    ADD COLUMN IF NOT EXISTS au1_code varchar(3),
    ADD COLUMN IF NOT EXISTS au2_code varchar(5);
CREATE INDEX IF NOT EXISTS idx_land_release_au2 ON public.land_release (au2_code);
