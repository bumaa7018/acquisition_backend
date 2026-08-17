-- Апп харахгүй байгаа өнчин ролиудыг устгана.
--
-- Урьд нь эрх бүр "<нөөц>_<түвшин>" (land_admin, compensation_employee ...)
-- гэсэн тусдаа роль болж задардаг байсан. Одоо систем нэг ролиор (admin,
-- senior_specialist, employee, decision_registration) ажилладаг тул эдгээр
-- үүсмэл ролиуд sd_system_role-оос хасагдсан — апп тэднийг ОГТ уншихаа больсон.
-- Гэвч эрх нь sd_role_permission-д хэвээр үлдсэн: sd_system_role-д мөр буцаж
-- нэмэгдэх агшинд тэр эрхүүд шууд сэргэх далд эрсдэлтэй.
--
-- sd_user_role-д FK байдаггүй тул хүүхэд мөрүүдийг гараар цэвэрлэнэ.
--
-- ХАМГААЛАЛТ: sd_role бол sdplatform-ын хуваалцсан хүснэгт. Өөр системийн
-- ролийг санамсаргүй устгахгүйн тулд (а) ЯМАР Ч системд харьяалагдаагүй БА
-- (б) энэ аппын үүсгэдэг нэрийн загварт яг таарсан мөрийг л устгана.

CREATE TEMP TABLE orphan_generated_roles ON COMMIT DROP AS
SELECT r.role_id
FROM sdplatform.sd_role r
WHERE NOT EXISTS (
        SELECT 1 FROM sdplatform.sd_system_role sr WHERE sr.role_id = r.role_id
      )
  AND r.name ~ ('^(land|compensation|decision|users|roles|permissions|hr|admin|audit)'
             || '_(admin|senior_specialist|employee|decision_registration'
             || '|professional_org|mika|finance_specialist)$');

DELETE FROM sdplatform.sd_user_role       WHERE role_id IN (SELECT role_id FROM orphan_generated_roles);
DELETE FROM sdplatform.sd_role_permission WHERE role_id IN (SELECT role_id FROM orphan_generated_roles);
DELETE FROM sdplatform.sd_menu_role       WHERE role_id IN (SELECT role_id FROM orphan_generated_roles);
DELETE FROM sdplatform.sd_role            WHERE role_id IN (SELECT role_id FROM orphan_generated_roles);
