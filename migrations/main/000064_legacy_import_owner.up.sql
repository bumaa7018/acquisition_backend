-- Хуучин мэдээллийн импортыг ахлах мэргэжилтэн ч хийдэг болсон. Хэрэглэгч зөвхөн
-- ӨӨРИЙН үүсгэсэн импортыг харж, устгадаг тул үүсгэсэн хэрэглэгчийн дугаарыг
-- (sd_user.user_id) хадгална. created_by нь харуулах нэр хэвээр.
--
-- Хуучин мөрүүдэд утга НӨХӨХГҮЙ (migration зөвхөн бүтэц): тэдгээрийг зөвхөн
-- системийн админ харна.
ALTER TABLE legacy_import_stage.session ADD COLUMN IF NOT EXISTS created_by_id integer;

CREATE INDEX IF NOT EXISTS legacy_import_session_owner
    ON legacy_import_stage.session (created_by_id);
