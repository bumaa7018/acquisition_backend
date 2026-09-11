-- ТӨЛӨВ СОЛИХОД ХАВСРАЛТ (ТАТГАЛЗСАН үед ЗААВАЛ).
--
-- "Татгалзсан" (4) төлөв нь нэгж талбарыг чөлөөлөлтөөс ГАРГАДАГ эцсийн
-- шийдвэр. Өмнө нь зөвхөн бичвэр шалтгаан (000008) үлддэг байсан ба тэр нь
-- маргаан гарахад хангалтгүй: татгалзлыг гэрчлэх баримт (иргэний өргөдөл,
-- шүүхийн шийдвэр, комиссын тэмдэглэл) хавсаргах газар байсангүй.
--
-- Хавсралт нь НЭГЖ ТАЛБАРЫН БАРИМТ (parcel_document) болж бүртгэгдэнэ —
-- ингэснээр "Хавсралт" хэсэгт бусад баримтын хамт харагдана. Энд зөвхөн
-- тухайн ШИЛЖИЛТИЙН холбоос үлдэнэ: явцын түүхэн дээр ямар баримтаар
-- татгалзсаныг шууд нээж харна.
--
-- ДАВТАЛТЫГ ТЭВЧИНЭ: `ADD COLUMN IF NOT EXISTS` нь идемпотент.
ALTER TABLE public.parcel_status_history
    ADD COLUMN IF NOT EXISTS attachment_url text DEFAULT ''::text NOT NULL,
    ADD COLUMN IF NOT EXISTS attachment_name text DEFAULT ''::text NOT NULL;

COMMENT ON COLUMN public.parcel_status_history.attachment_url IS
    'Тухайн шилжилтийн хавсралт (/api/files харьцангуй зам). Татгалзсан үед заавал; бусад төлөвт хоосон байж болно.';
COMMENT ON COLUMN public.parcel_status_history.attachment_name IS
    'Хавсралтын харагдах нэр. Файл нь нэгж талбарын баримт (parcel_document) болж мөн бүртгэгдсэн байна.';
