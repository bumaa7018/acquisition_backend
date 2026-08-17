-- Үнэлгээний (мэргэжлийн) байгууллагын бүртгэл.
--
-- Байгууллага өөрөө sdplatform.sd_organization дээр (HR лавлахтай ижил сан)
-- амьдарна — тэр хүснэгтэд ЗӨВХӨН (id, name) багана байдаг бөгөөд 000015-ын
-- зарчмаар бид sdplatform схемд багана НЭМЭХГҮЙ. Тиймээс регистр, тусгай
-- зөвшөөрөл зэрэг манай системд л хэрэгтэй талбаруудыг энэ буюу аппын ӨӨРИЙН
-- хүснэгтэд хадгална (000094_app_owned_auth_state-ийн адил загвар).
--
-- Ажилтнууд нь sdplatform-ын төрөлх бүтцээр бүртгэгдэнэ:
--   sd_organization → hr_department.organization → hr_employee.department_id
--   hr_employee.person_id → base.bs_person   (овог/нэр, регистр, утас, имэйл)
--   hr_employee.user_id   → sd_user          (нэвтрэх эрх; заавал бус)
-- Иймд ажилтны хүснэгт энд шинээр үүсэхгүй.

-- organization_id-д ГАДААД ТҮЛХҮҮР ЗОРИУД байхгүй: байгууллага өөрөө HR
-- лавлахтай ижил санд (auth DB) амьдардаг тул энэ сангаас FK тавих боломжгүй.
-- Байгууллага устахад профайлыг ValuationOrgRepo.Delete тусад нь устгана.
CREATE TABLE IF NOT EXISTS valuation_org_profile (
    organization_id    int4         PRIMARY KEY,
    short_name         varchar(250) NOT NULL DEFAULT '',
    register_no        varchar(20)  NOT NULL DEFAULT '',
    license_no         varchar(50)  NOT NULL DEFAULT '',
    license_issued_at  date,
    license_expires_at date,
    phone              varchar(50)  NOT NULL DEFAULT '',
    email              varchar(250) NOT NULL DEFAULT '',
    address            varchar(500) NOT NULL DEFAULT '',
    note               text         NOT NULL DEFAULT '',
    is_active          boolean      NOT NULL DEFAULT true,
    created_at         timestamptz  NOT NULL DEFAULT NOW(),
    updated_at         timestamptz  NOT NULL DEFAULT NOW()
);

-- Регистр давхардуулж бүртгэхээс сэргийлнэ (хоосон утга нь давхардаж болно —
-- регистр нь заавал биш).
CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_org_register
    ON valuation_org_profile (register_no)
    WHERE register_no <> '';

CREATE INDEX IF NOT EXISTS idx_valuation_org_active
    ON valuation_org_profile (is_active);
