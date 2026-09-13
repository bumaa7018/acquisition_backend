DROP INDEX IF EXISTS public.uq_valuation_org_employee_username;

ALTER TABLE public.valuation_org_employee
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS "password";

COMMENT ON COLUMN public.valuation_org_employee.user_id IS
    'Нэвтрэх эрх (sdplatform.sd_user.user_id). NULL бол ажилтан системд нэвтрэхгүй.';
