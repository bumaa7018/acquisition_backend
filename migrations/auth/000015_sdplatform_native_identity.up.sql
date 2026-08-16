-- Нэвтрэлт, эрх, HR лавлахыг sdplatform схемийн ТӨРӨЛХ бүтэц рүү бүрэн
-- шилжүүлнэ.
--
-- Зарчим: sdplatform / base схемийн хүснэгтүүдэд ЯМАР Ч өөрчлөлт оруулахгүй —
-- багана нэмэхгүй, төрөл өөрчлөхгүй, тэдгээрийг "тольдох" туслах хүснэгт
-- үүсгэхгүй. Ингэснээр энэ холболтыг жинхэнэ sdplatform өгөгдлийн сан руу
-- шууд залгахад, эсвэл өөр систем тэдгээр хүснэгтийг өөрчлөхөд систем хэвийн
-- ажиллана.
--
-- Үүнээс үүдэн:
--   * Бүх таних тэмдэг uuid-аас int4 (serial4) болно.
--   * 000012-д үүсгэсэн sd_user_api_meta / sd_role_api_meta /
--     sd_permission_api_meta болон HR-ийн meta хүснэгтүүд УСТАНА.
--   * Эрхийн загвар: sd_menu(+sd_menu_role) → цэс харагдах эсэх,
--     sd_role.role_type_code → НӨӨЦ (land, compensation, config ...),
--     sd_permission.name → ҮЙЛДЭЛ (read/create/update/delete).
--     Үр дүнгийн эрх = <role_type_code>:<permission.name> буюу "land:read".
--     Ролийн нэр нь "<нөөц>_<түвшин>" (land_admin) — түвшин нь JWT-д ордог тул
--     "admin" гэсэн шалгалтууд өмнөх шигээ ажиллана.
--   * hr_employee-д байгууллагын багана байхгүй тул ажилтны байгууллага нь
--     хэлтсээр дамжина (hr_employee.department_id → hr_department.organization).
--   * deleted_at багана байхгүй тул лавлахын устгал нь бодит DELETE; ажилтны
--     устгал нь hr_employee.is_active = 0.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ---------------------------------------------------------------------------
-- 1. Аппын өөрийн туслах хүснэгтүүд (sdplatform-д байдаггүй, бид үүсгэнэ)
--
-- Хэрэглэгчийн ID нь int4 болсон тул хуучин мөрүүд утгагүй; токенууд ямар ч
-- байсан хүчингүй болох тул дахин үүсгэнэ.
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS sdplatform.password_reset_tokens;
CREATE TABLE sdplatform.password_reset_tokens (
    id         uuid         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    int4         NOT NULL REFERENCES sdplatform.sd_user(user_id) ON DELETE CASCADE ON UPDATE CASCADE,
    token      varchar(255) NOT NULL UNIQUE,
    expires_at timestamptz  NOT NULL,
    used       boolean      NOT NULL DEFAULT false,
    created_at timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sdplatform_prt_token ON sdplatform.password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_sdplatform_prt_expires_at ON sdplatform.password_reset_tokens (expires_at);

DROP TABLE IF EXISTS sdplatform.revoked_tokens;
CREATE TABLE sdplatform.revoked_tokens (
    jti        varchar(255) PRIMARY KEY,
    user_id    int4         NOT NULL,
    token_type varchar(32)  NOT NULL,
    expires_at timestamptz  NOT NULL,
    revoked_at timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sdplatform_revoked_tokens_expires_at ON sdplatform.revoked_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_sdplatform_revoked_tokens_user_id ON sdplatform.revoked_tokens (user_id);

-- Токены эх цэг: нууц үг сэргээх / албадан гаргах үед өмнөх токеныг хүчингүй
-- болгоход. sd_user-т ийм багана байхгүй тул аппын өөрийн хүснэгтэд байна.
CREATE TABLE IF NOT EXISTS sdplatform.user_token_epoch (
    user_id    int4        PRIMARY KEY REFERENCES sdplatform.sd_user(user_id) ON DELETE CASCADE ON UPDATE CASCADE,
    valid_from timestamptz NOT NULL DEFAULT NOW()
);

-- Хуучин meta дэх токены эх цэгийг зөөнө (байгаа бол).
DO $$
BEGIN
    IF to_regclass('sdplatform.sd_user_api_meta') IS NOT NULL THEN
        INSERT INTO sdplatform.user_token_epoch (user_id, valid_from)
        SELECT m.user_id, m.tokens_valid_from
        FROM sdplatform.sd_user_api_meta m
        WHERE m.tokens_valid_from IS NOT NULL
        ON CONFLICT (user_id) DO NOTHING;
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 2. Эрхийн загварыг "нөөц:үйлдэл" → (роль=нөөц, эрх=үйлдэл) руу шилжүүлэх
-- ---------------------------------------------------------------------------

INSERT INTO sdplatform.sd_system (system_id, "name", description)
VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
ON CONFLICT (system_id) DO UPDATE SET
    "name" = EXCLUDED."name",
    description = EXCLUDED.description;

DO $$
DECLARE
    r         record;
    new_role  int4;
    new_perm  int4;
    role_name text;
BEGIN
    -- Хуучин загвар: эрхийн нэр "нөөц:үйлдэл", ролийн role_type_code хоосон.
    FOR r IN
        SELECT DISTINCT
               ro.role_id                        AS old_role_id,
               COALESCE(ro."name", '')           AS level,
               split_part(p."name", ':', 1)      AS resource,
               split_part(p."name", ':', 2)      AS action
        FROM sdplatform.sd_role_permission rp
        JOIN sdplatform.sd_role ro       ON ro.role_id = rp.role_id
        JOIN sdplatform.sd_permission p  ON p.permission_id = rp.permission_id
        WHERE p."name" LIKE '%:%'
          AND COALESCE(ro.role_type_code, '') = ''
          AND COALESCE(ro."name", '') <> ''
    LOOP
        role_name := r.resource || '_' || r.level;

        SELECT role_id INTO new_role FROM sdplatform.sd_role WHERE "name" = LEFT(role_name, 250);
        IF new_role IS NULL THEN
            INSERT INTO sdplatform.sd_role ("name", role_type_code, description, created_at, updated_at)
            VALUES (LEFT(role_name, 250), LEFT(r.resource, 45), r.level || ' — ' || r.resource, NOW(), NOW())
            RETURNING role_id INTO new_role;
        END IF;

        INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
        SELECT 10, new_role, NOW()
        WHERE NOT EXISTS (
            SELECT 1 FROM sdplatform.sd_system_role WHERE system_id = 10 AND role_id = new_role
        );

        SELECT permission_id INTO new_perm FROM sdplatform.sd_permission WHERE "name" = LEFT(r.action, 250);
        IF new_perm IS NULL THEN
            INSERT INTO sdplatform.sd_permission (code, "name", created_at)
            VALUES (LEFT(r.action, 45), LEFT(r.action, 250), NOW())
            RETURNING permission_id INTO new_perm;
        END IF;

        INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, created_at)
        SELECT new_role, new_perm, NOW()
        WHERE NOT EXISTS (
            SELECT 1 FROM sdplatform.sd_role_permission
            WHERE role_id = new_role AND permission_id = new_perm AND menu_id IS NULL
        );

        -- Хуучин роль эзэмшиж байсан бүх хэрэглэгчид шинэ ролийг олгоно.
        INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
        SELECT ur.user_id, new_role, NOW(), NOW()
        FROM sdplatform.sd_user_role ur
        WHERE ur.role_id = r.old_role_id
        ON CONFLICT DO NOTHING;
    END LOOP;
END $$;

-- Хуучин (нөөцгүй) ролиуд болон "нөөц:үйлдэл" хэлбэрийн эрхүүдийг цэвэрлэнэ.
DELETE FROM sdplatform.sd_role_permission rp
USING sdplatform.sd_permission p
WHERE rp.permission_id = p.permission_id AND p."name" LIKE '%:%';

DELETE FROM sdplatform.sd_permission WHERE "name" LIKE '%:%';

DELETE FROM sdplatform.sd_role
WHERE COALESCE(role_type_code, '') = ''
  AND EXISTS (
      SELECT 1 FROM sdplatform.sd_role nr
      WHERE nr.role_type_code IS NOT NULL
        AND nr."name" LIKE '%\_' || sdplatform.sd_role."name"
  );

-- ---------------------------------------------------------------------------
-- 3. HR лавлах: public → sdplatform (төрөлх бүтэц, int4 түлхүүр)
-- ---------------------------------------------------------------------------

DO $$
DECLARE
    r       record;
    new_id  int4;
    org_id  int4;
BEGIN
    IF to_regclass('public.organization') IS NULL THEN
        RETURN;
    END IF;

    -- Байгууллага (sd_organization: зөвхөн id, name)
    FOR r IN SELECT * FROM public.organization WHERE deleted_at IS NULL ORDER BY created_at, id LOOP
        IF NOT EXISTS (SELECT 1 FROM sdplatform.sd_organization WHERE "name" = LEFT(r.name, 250)) THEN
            INSERT INTO sdplatform.sd_organization ("name") VALUES (LEFT(r.name, 250));
        END IF;
    END LOOP;

    -- Албан тушаал
    FOR r IN SELECT * FROM public.position WHERE deleted_at IS NULL ORDER BY sort_order, name, id LOOP
        IF NOT EXISTS (SELECT 1 FROM sdplatform.hr_position WHERE "name" = LEFT(r.name, 250)) THEN
            INSERT INTO sdplatform.hr_position (code, "name", created_at, updated_at)
            VALUES (LEFT(NULLIF(r.code, ''), 45), LEFT(r.name, 250), r.created_at, r.updated_at);
        END IF;
    END LOOP;

    -- Хэлтэс
    FOR r IN
        SELECT d.*, o.name AS org_name
        FROM public.department d
        JOIN public.organization o ON o.id = d.organization_id
        WHERE d.deleted_at IS NULL
        ORDER BY d.sort_order, d.name, d.id
    LOOP
        SELECT id INTO org_id FROM sdplatform.sd_organization WHERE "name" = LEFT(r.org_name, 250) LIMIT 1;
        IF org_id IS NULL THEN
            CONTINUE;
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM sdplatform.hr_department
            WHERE organization = org_id AND "name" = LEFT(r.name, 250)
        ) THEN
            INSERT INTO sdplatform.hr_department (organization, code, "name", created_at, updated_at)
            VALUES (org_id, LEFT(NULLIF(r.code, ''), 45), LEFT(r.name, 250), r.created_at, r.updated_at);
        END IF;
    END LOOP;

    -- Хүн (bs_person). person_register нь varchar(10) тул уртыг таслана.
    FOR r IN SELECT * FROM public.person WHERE deleted_at IS NULL ORDER BY created_at, id LOOP
        IF EXISTS (SELECT 1 FROM base.bs_person WHERE person_register = LEFT(r.register_no, 10)) THEN
            CONTINUE;
        END IF;
        INSERT INTO base.bs_person
            ("type", "name", first_name, gender, date_of_birth, person_register,
             phone, email_address, address_town_or_local_name, civil_id, created_at, updated_at)
        VALUES (
            CASE r.person_type WHEN 'legal' THEN 2 ELSE 1 END,
            LEFT(COALESCE(CASE r.person_type WHEN 'legal' THEN r.legal_name ELSE r.last_name END, ''), 250),
            LEFT(CASE r.person_type WHEN 'legal' THEN NULL ELSE r.first_name END, 50),
            CASE r.gender WHEN 'M' THEN 1 WHEN 'F' THEN 2 ELSE NULL END,
            r.birth_date,
            LEFT(r.register_no, 10),
            LEFT(r.phone, 256),
            LEFT(r.email, 256),
            LEFT(r.address, 256),
            CASE WHEN r.civil_id ~ '^[0-9]+$' THEN r.civil_id::int8 ELSE NULL END,
            r.created_at, r.updated_at
        );
    END LOOP;

    -- Ажилтан. hr_employee-д байгууллагын багана байхгүй — хэлтсээр дамжина.
    FOR r IN
        SELECT e.*, p.register_no, pos.name AS position_name, d.name AS department_name, o.name AS org_name
        FROM public.employee e
        JOIN public.person p        ON p.id = e.person_id
        JOIN public.position pos    ON pos.id = e.position_id
        JOIN public.organization o  ON o.id = e.organization_id
        LEFT JOIN public.department d ON d.id = e.department_id
        WHERE e.deleted_at IS NULL
        ORDER BY e.created_at, e.id
    LOOP
        SELECT person_id INTO new_id FROM base.bs_person WHERE person_register = LEFT(r.register_no, 10) LIMIT 1;
        IF new_id IS NULL THEN
            CONTINUE;
        END IF;
        IF EXISTS (SELECT 1 FROM sdplatform.hr_employee WHERE person_id = new_id AND COALESCE(is_active, 1) <> 0) THEN
            CONTINUE;
        END IF;
        INSERT INTO sdplatform.hr_employee (
            person_id, department_id, position_id, firstname, lastname,
            email, mobile_phone, hired_date, terminated_date, is_active, created_at, updated_at
        )
        SELECT
            new_id,
            (SELECT dd.department_id
               FROM sdplatform.hr_department dd
               JOIN sdplatform.sd_organization oo ON oo.id = dd.organization
              WHERE dd."name" = LEFT(r.department_name, 250) AND oo."name" = LEFT(r.org_name, 250)
              LIMIT 1),
            (SELECT pp.position_id FROM sdplatform.hr_position pp WHERE pp."name" = LEFT(r.position_name, 250) LIMIT 1),
            LEFT(bp.first_name, 250), LEFT(bp."name", 250),
            LEFT(r.work_email, 250), LEFT(r.work_phone, 50),
            r.hired_at, r.released_at,
            CASE WHEN r.is_active THEN 1 ELSE 0 END,
            r.created_at, r.updated_at
        FROM base.bs_person bp
        WHERE bp.person_id = new_id;
    END LOOP;
END $$;

-- Хэрэглэгч ↔ ажилтны холбоос нь hr_employee.user_id дээр байна.
DO $$
BEGIN
    IF to_regclass('sdplatform.sd_user_api_meta') IS NOT NULL
       AND to_regclass('sdplatform.hr_employee_api_meta') IS NOT NULL THEN
        UPDATE sdplatform.hr_employee e
        SET user_id = um.user_id
        FROM sdplatform.hr_employee_api_meta em
        JOIN sdplatform.sd_user_api_meta um ON um.employee_uuid = em.api_uuid
        WHERE e.employee_id = em.employee_id AND e.user_id IS DISTINCT FROM um.user_id;
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 4. Meta хүснэгтүүд болон public дахь HR лавлахыг устгах
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS sdplatform.hr_employee_api_meta;
DROP TABLE IF EXISTS base.bs_person_api_meta;
DROP TABLE IF EXISTS sdplatform.hr_department_api_meta;
DROP TABLE IF EXISTS sdplatform.hr_position_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_organization_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_user_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_role_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_permission_api_meta;

DROP TABLE IF EXISTS public.employee;
DROP TABLE IF EXISTS public.person;
DROP TABLE IF EXISTS public.department;
DROP TABLE IF EXISTS public.position;
DROP TABLE IF EXISTS public.organization;
