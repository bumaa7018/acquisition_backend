-- НЭГЖ ТАЛБАРУУДЫН БАЙРШЛЫН ДАВХАРДАЛ.
--
-- Асуудал: ГУС-аас татагдсан нэгж талбарууд хоорондоо ОРОН ЗАЙН хувьд
-- давхцаж ирдэг (хуучин/шинэ хэмжилт, залруулга хийгдээгүй бүртгэл, нэг газар
-- дээр хоёр дугаар). Давхардалтай хэвээр чөлөөлөлтийг үргэлжлүүлбэл нэг газрын
-- нөхөх олговор ХОЁР УДАА бодогдох эрсдэлтэй.
--
-- Иймд давхардлыг нэгж талбар татах мөчид тооцож ЭНД бүртгэнэ:
--   * "Хээрийн судалгаа" төлөв рүү шилжихийг хаана (давхардал арилтал);
--   * нэгж талбарын дэлгэрэнгүйд улаанаар харуулна;
--   * жагсаалтын шүүлтүүрээр (давхардалтай эсэх) хайна.
--
-- ХОЁР ЧИГЛЭЛЭЭР хадгална (A→B ба B→A): нэг нэгж талбарын давхардлыг олоход
-- нэг л мөрийн шүүлт хийгддэг (OR-гүй), шүүлтүүр нь тэгш хэмтэй болно.
--
-- Дахин тооцоолохдоо тухайн чөлөөлөлтийн мөрүүд БҮГД устгагдаад шинээр
-- бичигддэг тул хуучин (арилсан) давхардал үлдэхгүй.
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: гадаад/шалгах хязгаарлалтууд БҮГД `CREATE TABLE IF NOT
-- EXISTS`-ийн ДОТОР зарлагдсан. `ALTER TABLE ... ADD CONSTRAINT` нь
-- `IF NOT EXISTS`-ийг дэмждэггүй тул тусад нь бичвэл миграци хоёр дахь удаа
-- ажиллахад "already exists"-ээр унаж, schema_migrations нь dirty=1 болж
-- апп restart бүрд уначихдаг (cmd/api/migrate.go-ийн тайлбарыг үз).
CREATE TABLE IF NOT EXISTS public.parcel_overlap (
    parcel_id uuid NOT NULL REFERENCES public.parcel(id) ON DELETE CASCADE,
    other_parcel_id uuid NOT NULL REFERENCES public.parcel(id) ON DELETE CASCADE,
    acquisition_id uuid NOT NULL REFERENCES public.land_acquisition(id) ON DELETE CASCADE,
    -- Уншихад зориулсан хуулбар: жагсаалт/дэлгэрэнгүй харуулахад parcel руу
    -- дахин JOIN хийхгүйгээр дугаарыг шууд гаргана.
    other_parcel_code character varying(12) DEFAULT ''::character varying NOT NULL,
    -- Давхцах ХЭСГИЙН талбай. ГУС-тай ижил UTM аргаар (public.calculate_area_utm)
    -- бодогдоно — ST_Area(::geography) ХЭРЭГЛЭХГҮЙ (000006 миграцийг үз).
    overlap_area_m2 numeric(18,4) DEFAULT 0 NOT NULL,
    detected_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT parcel_overlap_pkey PRIMARY KEY (parcel_id, other_parcel_id),
    -- Нэгж талбар өөртэйгөө давхцаж БОЛОХГҮЙ (тооцооллын алдааг барина).
    CONSTRAINT parcel_overlap_distinct CHECK (parcel_id <> other_parcel_id)
);

-- Чөлөөлөлтийн бүх давхардлыг унших / дахин тооцохын өмнө устгах.
CREATE INDEX IF NOT EXISTS idx_parcel_overlap_acquisition
    ON public.parcel_overlap (acquisition_id);

COMMENT ON TABLE public.parcel_overlap IS
    'Нэгж талбаруудын байршлын давхардал (хоёр чиглэлээр). Нэгж талбар татах мөчид тооцогдоно.';
COMMENT ON COLUMN public.parcel_overlap.overlap_area_m2 IS
    'Давхцах хэсгийн талбай, м² (public.calculate_area_utm).';
