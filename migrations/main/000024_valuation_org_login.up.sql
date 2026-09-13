-- Үнэлгээний байгууллагын ажилтны НЭВТРЭХ ЭРХ appdb-д шилжив.
--
-- ӨМНӨ: нэвтрэх эрх нь sdplatform.sd_user-т үүсдэг байсан — seeder-ээс ч,
-- байгууллага бүртгэх API-аас ч (ValuationOrgRepo.ensureLogins). authdb нь
-- ажиллаж байгаа ГУС-ийн сан тул апп тэнд хэрэглэгч үүсгэх/идэвхгүй болгох
-- ёсгүй.
--
-- ОДОО: ажилтны мөр ӨӨРӨӨ нэвтрэх нэгж болно. valuation_org_employee.id нь
-- эдгээр хэрэглэгчийн user_id (user_source='valuation_org'). Эрхийн хувьд юу ч
-- өөрчлөгдөхгүй: эдгээр хэрэглэгч sdplatform-ийн роль/эрх ОГТ хэрэглэдэггүй,
-- зөвхөн токены `valuation_org` тэмдгээр хязгаарлагддаг.
--
-- ЖИЧ: user_id багана ХЭВЭЭР үлдэнэ (хуучин sd_user холбоос). Шинэ эрх тэнд
-- үүсэхээ больсон — уншихад л хэрэглэнэ.
ALTER TABLE public.valuation_org_employee
    ADD COLUMN IF NOT EXISTS username character varying(50),
    ADD COLUMN IF NOT EXISTS "password" character varying(150);

-- Нэвтрэх нэр ИДЭВХТЭЙ ажилтнуудын дунд давхардахгүй. Идэвхгүй мөр нэрийг
-- эзэлдэггүй тул ажилтныг хасаад дахин бүртгэхэд саад болохгүй
-- (uq_valuation_org_employee_user-тай ижил зарчим).
CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_org_employee_username
    ON public.valuation_org_employee USING btree (lower((username)::text))
    WHERE (username IS NOT NULL AND is_active);

COMMENT ON COLUMN public.valuation_org_employee.username IS
    'Нэвтрэх нэр (appdb-д хадгалагдана, sd_user-т БИШ). NULL бол нэвтрэх эрхгүй.';
COMMENT ON COLUMN public.valuation_org_employee."password" IS
    'bcrypt hash. Нэвтрэлт user_source=''valuation_org''-оор энэ хүснэгтээс шалгагдана.';
COMMENT ON COLUMN public.valuation_org_employee.user_id IS
    'ХУУЧИН sdplatform.sd_user.user_id холбоос. Шинэ эрх энд үүсэхгүй — username/password-ыг үз.';
