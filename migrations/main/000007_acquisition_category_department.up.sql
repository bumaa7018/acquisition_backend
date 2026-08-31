-- Ерөнхий ангилалд ХАРИУЦСАН АЛБА (хэлтэс) холбоно.
--
-- Алба нь ГУС-ийн бүртгэл (authdb.sdplatform.hr_department) бөгөөд appdb-ээс
-- ӨӨР САНД байдаг тул FOREIGN KEY тавих БОЛОМЖГҮЙ. Иймд зөвхөн дугаарыг нь
-- хадгалж, нэрийг дэлгэц дээр /departments жагсаалтаас тааруулна —
-- land_acquisition_assignee.user_id-тай ижил зарчим.
--
-- NULL = хариуцсан алба тодорхойлоогүй (хуучин мөрүүд бүгд ийм байна).
ALTER TABLE public.acquisition_category
    ADD COLUMN IF NOT EXISTS department_id integer;

COMMENT ON COLUMN public.acquisition_category.department_id IS
    'Хариуцсан албаны дугаар (authdb.sdplatform.hr_department.department_id). Өөр сан тул FK байхгүй.';
