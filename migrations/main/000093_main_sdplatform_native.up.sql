-- Үндсэн (main) DB: гадаад хэрэглэгчид (мэргэжлийн байгууллага, МИКА, санхүү)
-- мөн sdplatform схем дээр амьдарна — auth DB-тэй ЯГ ижил repo код ажилладаг.
--
-- Мөн энэ DB-д хэрэглэгчийн ID заадаг багануудыг uuid → int4 болгоно
-- (sdplatform-д uuid байхгүй).

CREATE SCHEMA IF NOT EXISTS sdplatform;

-- Хүснэгтүүдийн тодорхойлолт нь sdplatform-ынхтой ЯГ ижил (нэмэлт багана,
-- нэмэлт индекс байхгүй) тул жинхэнэ sdplatform DB рүү залгахад
-- CREATE ... IF NOT EXISTS нь юу ч хийхгүй өнгөрнө.
CREATE TABLE IF NOT EXISTS sdplatform.sd_user (
    user_id        serial4 NOT NULL,
    username       varchar(50)  NULL,
    email          varchar(250) NULL,
    "password"     varchar(150) NULL,
    firstname      varchar(50)  NULL,
    lastname       varchar(50)  NULL,
    photo          varchar(250) NULL,
    remember_token varchar(100) NULL,
    created_at     date      NULL,
    updated_at     timestamp NULL,
    deleted_at     timestamp NULL,
    created_by     int4 NULL,
    updated_by     int4 NULL,
    last_active    timestamptz NULL,
    gis_user_real  varchar(20) NULL DEFAULT NULL::character varying,
    is_active      bool NULL DEFAULT false,
    CONSTRAINT sd_user_pkey PRIMARY KEY (user_id)
);

CREATE SEQUENCE IF NOT EXISTS sdplatform.sd_role_id_seq;

CREATE TABLE IF NOT EXISTS sdplatform.sd_role (
    role_id        int4 NOT NULL DEFAULT nextval('sdplatform.sd_role_id_seq'::regclass),
    code           varchar(10)  NULL DEFAULT NULL::character varying,
    "name"         varchar(250) NULL DEFAULT NULL::character varying,
    description    text NULL,
    role_type_code varchar(45)  NULL DEFAULT NULL::character varying,
    created_at     timestamp NULL,
    updated_at     timestamp NULL,
    created_by     int4 NULL,
    updated_by     int4 NULL,
    editor_user    int4 NULL,
    CONSTRAINT sd_role_pkey PRIMARY KEY (role_id),
    CONSTRAINT sd_role_uk UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_permission (
    permission_id serial4 NOT NULL,
    code       varchar(45)  NULL DEFAULT NULL::character varying,
    "name"     varchar(250) NULL DEFAULT NULL::character varying,
    created_at timestamp NULL,
    updated_at timestamp NULL,
    created_by int4 NULL,
    updated_by int4 NULL,
    CONSTRAINT sd_permission_pkey PRIMARY KEY (permission_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_user_role (
    user_id    int4 NOT NULL,
    role_id    int4 NOT NULL,
    created_at timestamp NULL,
    updated_at timestamp NULL,
    created_by int4 NULL,
    updated_by int4 NULL,
    CONSTRAINT sd_user_role_pkey PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_system (
    system_id   serial4 NOT NULL,
    "name"      varchar(250) NULL DEFAULT NULL::character varying,
    description text NULL,
    CONSTRAINT pk_sd_system PRIMARY KEY (system_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_system_role (
    system_role_id serial4 NOT NULL,
    system_id  int4 NULL,
    role_id    int4 NULL,
    created_at timestamp NULL,
    updated_at timestamp NULL,
    CONSTRAINT pk_sd_system_role PRIMARY KEY (system_role_id),
    CONSTRAINT fk_sd_system_role_sd_role FOREIGN KEY (role_id) REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_sd_system_role_sd_system FOREIGN KEY (system_id) REFERENCES sdplatform.sd_system(system_id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_menu (
    menu_id     serial4 NOT NULL,
    code        varchar(32)  NULL DEFAULT NULL::character varying,
    "name"      varchar(250) NULL DEFAULT NULL::character varying,
    parent_id   int4 NULL,
    menu_url    varchar(250) NULL DEFAULT NULL::character varying,
    sort_order  int4 NULL,
    created_at  timestamp NULL,
    updated_at  timestamp NULL,
    created_by  int4 NULL,
    updated_by  int4 NULL,
    menu_icon   varchar(150) NULL DEFAULT NULL::character varying,
    parent_code varchar(20) NULL DEFAULT NULL::character varying,
    CONSTRAINT sd_menu_pkey PRIMARY KEY (menu_id),
    CONSTRAINT sd_menuu_uk UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_role_permission (
    role_permission_id serial4 NOT NULL,
    role_id       int4 NULL,
    permission_id int4 NULL,
    menu_id       int4 NULL,
    created_at    timestamp NULL,
    updated_at    timestamp NULL,
    CONSTRAINT sd_role_permission_pkey PRIMARY KEY (role_permission_id),
    CONSTRAINT sd_role_permission_fkey1 FOREIGN KEY (menu_id) REFERENCES sdplatform.sd_menu(menu_id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT sd_role_permission_fkey2 FOREIGN KEY (permission_id) REFERENCES sdplatform.sd_permission(permission_id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT sd_role_permission_fkey3 FOREIGN KEY (role_id) REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- HR хүснэгтүүд: main DB-д ажилтны бүртгэл байхгүй ч UserRepo нь бүх DB дээр
-- ИЖИЛ SQL-ээр ажилладаг (sd_user → hr_employee → hr_position). Тиймээс эдгээр
-- нь энд ХООСОН боловч байх ёстой. Тодорхойлолт нь sdplatform-ынхтой яг ижил.
CREATE SCHEMA IF NOT EXISTS base;
CREATE SCHEMA IF NOT EXISTS codelists;

CREATE TABLE IF NOT EXISTS codelists.cl_person_type (
    code           int4        NOT NULL,
    description    varchar(75) NOT NULL,
    description_en varchar(75) NULL,
    CONSTRAINT cl_person_type_description_en_key UNIQUE (description_en),
    CONSTRAINT cl_person_type_description_key UNIQUE (description),
    CONSTRAINT cl_person_type_pkey PRIMARY KEY (code)
);
INSERT INTO codelists.cl_person_type (code, description, description_en)
VALUES (1, 'Иргэн', 'citizen'), (2, 'Хуулийн этгээд', 'legal')
ON CONFLICT (code) DO NOTHING;

CREATE SEQUENCE IF NOT EXISTS base.ps_person_person_id_seq;

CREATE TABLE IF NOT EXISTS base.bs_person (
    person_id       int4 NOT NULL DEFAULT nextval('base.ps_person_person_id_seq'::regclass),
    "type"          int4 NOT NULL,
    "name"          varchar(250) NOT NULL,
    middle_name     varchar(50) NULL,
    first_name      varchar(50) NULL,
    gender          int4 NULL,
    date_of_birth   date NULL,
    person_register varchar(10) NOT NULL,
    phone           varchar(256) NULL,
    mobile_phone    varchar(256) NULL,
    email_address   varchar(256) NULL,
    address_town_or_local_name varchar(256) NULL,
    created_at      timestamp NULL,
    updated_at      timestamp NULL,
    civil_id        int8 NULL,
    CONSTRAINT ps_person_pkey PRIMARY KEY (person_id),
    CONSTRAINT bs_person_type_fkey FOREIGN KEY ("type") REFERENCES codelists.cl_person_type(code) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_organization (
    id     serial4 NOT NULL,
    "name" varchar(250) NULL,
    CONSTRAINT sd_organization_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_position (
    position_id serial4 NOT NULL,
    code        varchar(45)  NULL DEFAULT NULL::character varying,
    "name"      varchar(250) NULL DEFAULT NULL::character varying,
    created_at  timestamp NULL,
    updated_at  timestamp NULL,
    organization int4 NULL,
    department   int4 NULL,
    is_meeting   bool NOT NULL DEFAULT false,
    CONSTRAINT hr_employee_uk UNIQUE (code),
    CONSTRAINT hr_position_pkey PRIMARY KEY (position_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_department (
    department_id serial4 NOT NULL,
    code       varchar(45)  NULL DEFAULT NULL::character varying,
    "name"     varchar(250) NULL DEFAULT NULL::character varying,
    parent_id  int4 NULL,
    created_at timestamp NULL,
    updated_at timestamp NULL,
    organization int4 NULL,
    CONSTRAINT hr_department_pkey PRIMARY KEY (department_id),
    CONSTRAINT hr_department_fkey1 FOREIGN KEY (parent_id) REFERENCES sdplatform.hr_department(department_id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT sd_organization_fk FOREIGN KEY (organization) REFERENCES sdplatform.sd_organization(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_employee (
    employee_id serial4 NOT NULL,
    firstname   varchar(250) NULL DEFAULT NULL::character varying,
    lastname    varchar(250) NULL DEFAULT NULL::character varying,
    mobile_phone varchar(50) NULL DEFAULT NULL::character varying,
    department_id int4 NULL,
    position_id   int4 NULL,
    user_id       int4 NULL,
    created_at    timestamp NULL,
    updated_at    timestamp NULL,
    hired_date      date NULL,
    terminated_date date NULL,
    is_active   int4 NULL,
    email       varchar(250) NULL DEFAULT NULL::character varying,
    person_id   int4 NULL,
    CONSTRAINT hr_employee_pkey PRIMARY KEY (employee_id),
    CONSTRAINT hr_employee_fkey1 FOREIGN KEY (person_id) REFERENCES base.bs_person(person_id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT hr_employee_fkey2 FOREIGN KEY (user_id) REFERENCES sdplatform.sd_user(user_id) ON DELETE SET NULL ON UPDATE CASCADE
);

INSERT INTO sdplatform.sd_system (system_id, "name", description)
VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
ON CONFLICT (system_id) DO UPDATE SET
    "name" = EXCLUDED."name",
    description = EXCLUDED.description;

-- ---------------------------------------------------------------------------
-- public.users/roles/permissions → sdplatform (нөөц:үйлдэл → роль/эрх)
-- ---------------------------------------------------------------------------

DO $$
DECLARE
    r         record;
    new_user  int4;
    new_role  int4;
    new_perm  int4;
    role_name text;
BEGIN
    IF to_regclass('public.users') IS NOT NULL THEN
        FOR r IN SELECT * FROM public.users ORDER BY created_at, id LOOP
            IF EXISTS (SELECT 1 FROM sdplatform.sd_user WHERE email = LEFT(r.email, 250)) THEN
                CONTINUE;
            END IF;
            INSERT INTO sdplatform.sd_user
                (username, email, "password", firstname, lastname, created_at, updated_at, deleted_at, is_active)
            VALUES (
                LEFT(r.username, 50), LEFT(r.email, 250), LEFT(r.password, 150),
                LEFT(r.first_name, 50), LEFT(r.last_name, 50),
                r.created_at::date, r.updated_at::timestamp, r.deleted_at::timestamp, r.is_active
            )
            RETURNING user_id INTO new_user;
        END LOOP;
    END IF;

    IF to_regclass('public.role_permissions') IS NOT NULL THEN
        FOR r IN
            SELECT DISTINCT ro.name AS level,
                   split_part(p.name, ':', 1) AS resource,
                   split_part(p.name, ':', 2) AS action,
                   ro.id AS old_role_id
            FROM public.role_permissions rp
            JOIN public.roles ro      ON ro.id = rp.role_id
            JOIN public.permissions p ON p.id = rp.permission_id
            WHERE p.name LIKE '%:%'
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

            -- Тухайн хуучин роль эзэмшиж байсан хэрэглэгчид шинэ ролийг олгоно.
            INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
            SELECT su.user_id, new_role, NOW(), NOW()
            FROM public.user_roles ur
            JOIN public.users u  ON u.id = ur.user_id
            JOIN sdplatform.sd_user su ON su.email = LEFT(u.email, 250)
            WHERE ur.role_id = r.old_role_id
            ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;
END $$;


-- Аль хэдийн sdplatform-д шилжсэн боловч ХУУЧИН хэлбэртэй (эрхийн нэр
-- "нөөц:үйлдэл", ролийн role_type_code хоосон) өгөгдлийг байрандаа хөрвүүлнэ.
-- Шинэ DB дээр энэ нь юу ч хийхгүй.
DO $$
DECLARE
    r         record;
    new_role  int4;
    new_perm  int4;
    role_name text;
BEGIN
    FOR r IN
        SELECT DISTINCT
               ro.role_id                   AS old_role_id,
               COALESCE(ro."name", '')      AS level,
               split_part(p."name", ':', 1) AS resource,
               split_part(p."name", ':', 2) AS action
        FROM sdplatform.sd_role_permission rp
        JOIN sdplatform.sd_role ro      ON ro.role_id = rp.role_id
        JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
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

        INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
        SELECT ur.user_id, new_role, NOW(), NOW()
        FROM sdplatform.sd_user_role ur
        WHERE ur.role_id = r.old_role_id
        ON CONFLICT DO NOTHING;
    END LOOP;
END $$;

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

-- sd_user_role-д гадаад түлхүүр байдаггүй тул устгасан хуучин ролиудын
-- өнчин мөрүүдийг гараар цэвэрлэнэ (эс тэгвээс sequence дахин ашиглахад
-- буруу роль хавсрах эрсдэлтэй).
DELETE FROM sdplatform.sd_user_role ur
WHERE NOT EXISTS (SELECT 1 FROM sdplatform.sd_role r WHERE r.role_id = ur.role_id);

-- Хуучин *_api_meta хүснэгтүүд (000092-оор үүсээгүй бол байхгүй).
DROP TABLE IF EXISTS sdplatform.sd_user_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_role_api_meta;
DROP TABLE IF EXISTS sdplatform.sd_permission_api_meta;

-- ---------------------------------------------------------------------------
-- Хэрэглэгчийн ID заадаг багануудыг uuid → int4
-- ---------------------------------------------------------------------------

-- Хуучин uuid утгууд шинэ int4 хэрэглэгчид харгалзахгүй тул хөрвүүлэх үед
-- бүгдийг NULL болгоно (гадаад холбоос алдагдана).
--
-- ЗААВАЛ ЭХЛЭЭД мөрийг устгах хүснэгтүүд: доорх багануудын зарим нь NOT NULL
-- (land_acquisition_assignee.user_id нь бүр PK-ийн хэсэг) тул "USING NULL"
-- шууд хийвэл өгөгдөлтэй DB дээр migration унана. Утга нь ямар ч тохиолдолд
-- алдагдах тул мөрийг нь урьдчилан цэвэрлэх нь цорын ганц зөв арга.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'land_acquisition_assignee'
          AND column_name = 'user_id' AND data_type = 'uuid'
    ) THEN
        DELETE FROM public.land_acquisition_assignee;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'notification'
          AND column_name = 'recipient_id' AND data_type = 'uuid'
    ) THEN
        DELETE FROM public.notification;
    END IF;
END $$;

DO $$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT * FROM (VALUES
            ('audit_log', 'actor_id'),
            ('drone_image', 'uploaded_by'),
            ('land_acquisition', 'professional_org_id'),
            ('land_acquisition_assignee', 'assigned_by'),
            ('land_acquisition_assignee', 'user_id'),
            ('notification', 'actor_id'),
            ('notification', 'recipient_id'),
            ('parcel', 'independent_org_id')
        ) AS t(tbl, col)
    LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = c.tbl
              AND column_name = c.col AND data_type = 'uuid'
        ) THEN
            EXECUTE format('ALTER TABLE public.%I ALTER COLUMN %I DROP DEFAULT', c.tbl, c.col);
            EXECUTE format('ALTER TABLE public.%I ALTER COLUMN %I TYPE int4 USING NULL', c.tbl, c.col);
        END IF;
    END LOOP;
END $$;

DROP TABLE IF EXISTS public.role_permissions;
DROP TABLE IF EXISTS public.user_roles;
DROP TABLE IF EXISTS public.permissions;
DROP TABLE IF EXISTS public.roles;
DROP TABLE IF EXISTS public.users;
