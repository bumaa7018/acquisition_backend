-- parcel.db_changed: BOOLEAN → TEXT
--
-- "Мэдээллийн санд өөрчлөлт орсон эсэх" нь тийм/үгүй биш, ТАЙЛБАР талбар:
-- нөхөх олговрын тайлангийн эксэл дээр "устгагдсан", "нэгж талбарын дугаар
-- өөрчлөгдсөн", "хэмжээ өөрчлөгдсөн" гэх мэт чөлөөт текстээр ирдэг. Boolean
-- болгож хураангуйлахад ЯМАР өөрчлөлт орсон нь алдагддаг тул текст болгож,
-- эксэлийн тайлбарыг шууд хадгална. Хэрэглэгч мөн гараас засна.
--
-- Хөрвүүлэлт: true → 'Тийм' (өмнө нь зөвхөн ийм л мэдээлэл байсан),
-- false/NULL → '' (хоосон = өөрчлөлт ороогүй).
--
-- Багана нь аль хэдийн text болсон бол (дахин ажиллуулалт) алхам бүхэлдээ
-- алгасагдана — ALTER ... TYPE нь өөрөө idempotent БИШ.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'parcel'
           AND column_name = 'db_changed'
           AND data_type = 'boolean'
    ) THEN
        ALTER TABLE parcel ALTER COLUMN db_changed DROP DEFAULT;
        ALTER TABLE parcel ALTER COLUMN db_changed DROP NOT NULL;
        ALTER TABLE parcel
            ALTER COLUMN db_changed TYPE text
            USING (CASE WHEN db_changed THEN 'Тийм' ELSE '' END);
        ALTER TABLE parcel ALTER COLUMN db_changed SET DEFAULT '';
    END IF;
END $$;

COMMENT ON COLUMN parcel.db_changed IS
    'Мэдээллийн санд орсон өөрчлөлтийн ТАЙЛБАР (чөлөөт текст). Хоосон = өөрчлөлт ороогүй.';
