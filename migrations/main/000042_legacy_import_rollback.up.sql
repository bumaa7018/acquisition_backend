-- ХУУЧИН МЭДЭЭЛЛИЙН ИМПОРТЫГ «ХАСАХ» (бүрэн буцаах).
--
-- Импорт ҮҮСГЭСЭН чөлөөлөлт бүр аль импортоос гарсныг legacy_import_id-д
-- хадгална. NULL биш = «deletable»: админ «Хуучин мэдээлэл оруулах» жагсаалтаас
-- «Хасах» дарж баталгаажуулбал ТУХАЙН чөлөөлөлт ба түүнтэй холбоотой БҮХ
-- мэдээлэл (импортын дараа нэмэгдсэн ч) бүрэн устгагдана. Гараар үүсгэсэн эсвэл
-- өөр импортын чөлөөлөлт NULL тул хэзээ ч хамрагдахгүй.
--
-- RESTRICT: чөлөөлөлт нь үлдэж байхад импортын бүртгэлийг устгах боломжгүй —
-- «deletable» тэмдэг хэзээ ч алдагдахгүй.
ALTER TABLE public.land_acquisition
    ADD COLUMN IF NOT EXISTS legacy_import_id uuid
        REFERENCES legacy_import_stage.session(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS land_acquisition_legacy_import_id
    ON public.land_acquisition (legacy_import_id) WHERE legacy_import_id IS NOT NULL;
COMMENT ON COLUMN public.land_acquisition.legacy_import_id IS
    'Энэ чөлөөлөлтийг ҮҮСГЭСЭН хуучин мэдээллийн импорт. NULL биш бол «Хасах»-аар бүрэн устгах боломжтой.';

-- Импорт шинээр ҮҮСГЭСЭН дундын лавлах мөрүүд (төлөвлөгөө, эзэмшигч,
-- захирамж). Эдгээр нь өөр чөлөөлөлтөд хэрэглэгдэж болох тул «Хасах» үед
-- ЗӨВХӨН өөр газар хэрэглэгдээгүй бол устгана. Импорт өмнө нь байсан мөрийг
-- ХЭЗЭЭ Ч өөрчлөхгүй, энд ч бүртгэхгүй.
CREATE TABLE IF NOT EXISTS legacy_import_stage.owned (
    session_id uuid NOT NULL REFERENCES legacy_import_stage.session(id) ON DELETE RESTRICT,
    kind text NOT NULL,
    ref_id uuid NOT NULL,
    PRIMARY KEY (session_id, kind, ref_id),
    CONSTRAINT legacy_import_owned_kind CHECK (kind IN ('plan', 'right_holder', 'decision_draft'))
);

ALTER TABLE legacy_import_stage.session ADD COLUMN IF NOT EXISTS rollback jsonb;
ALTER TABLE legacy_import_stage.session DROP CONSTRAINT IF EXISTS legacy_import_session_status;
ALTER TABLE legacy_import_stage.session ADD CONSTRAINT legacy_import_session_status
    CHECK (status IN ('draft', 'committing', 'committed', 'failed', 'rolled_back'));
