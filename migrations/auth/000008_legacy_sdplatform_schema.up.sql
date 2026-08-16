CREATE SCHEMA IF NOT EXISTS sdplatform;
CREATE SCHEMA IF NOT EXISTS base;
CREATE SCHEMA IF NOT EXISTS codelists;
CREATE SCHEMA IF NOT EXISTS audit;

CREATE OR REPLACE FUNCTION base.log_changes()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION base.notify_event_all()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION audit.if_modified_func()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END $$;

CREATE TABLE IF NOT EXISTS codelists.cl_person_type (
  code int4 NOT NULL,
  description varchar(75) NOT NULL,
  description_en varchar(75) NULL,
  CONSTRAINT cl_person_type_description_en_key UNIQUE (description_en),
  CONSTRAINT cl_person_type_description_key UNIQUE (description),
  CONSTRAINT cl_person_type_pkey PRIMARY KEY (code)
);

CREATE TABLE IF NOT EXISTS codelists.cl_legal_entity_type (
  id int4 NOT NULL,
  description varchar(250) NULL,
  CONSTRAINT cl_legal_entity_type_pkey PRIMARY KEY (id)
);

INSERT INTO codelists.cl_person_type (code, description, description_en)
VALUES (1, 'Иргэн', 'citizen'), (2, 'Хуулийн этгээд', 'legal')
ON CONFLICT (code) DO NOTHING;

CREATE SEQUENCE IF NOT EXISTS base.ps_person_person_id_seq;

CREATE TABLE IF NOT EXISTS base.bs_person (
  person_id int4 NOT NULL DEFAULT nextval('base.ps_person_person_id_seq'::regclass),
  "type" int4 NOT NULL,
  "name" varchar(250) NOT NULL,
  middle_name varchar(50) NULL,
  first_name varchar(50) NULL,
  gender int4 NULL,
  date_of_birth date NULL,
  contact_surname varchar(50) NULL,
  contact_first_name varchar(50) NULL,
  contact_position varchar(50) NULL,
  person_register varchar(10) NOT NULL,
  state_registration_no varchar(20) NULL,
  bank int4 NULL,
  bank_account_no varchar(20) NULL,
  phone varchar(256) NULL,
  mobile_phone varchar(256) NULL,
  email_address varchar(256) NULL,
  website varchar(60) NULL,
  address_town_or_local_name varchar(256) NULL,
  address_neighbourhood varchar(250) NULL,
  address_street_name varchar(250) NULL,
  address_khaskhaa varchar(50) NULL,
  address_building_no varchar(10) NULL,
  address_entrance_no varchar(10) NULL,
  address_apartment_no varchar(10) NULL,
  address_au_level1 varchar(3) NULL,
  address_au_level2 varchar(5) NULL,
  address_au_level3 varchar(8) NULL,
  address_au_khoroolol int4 NULL,
  country int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  parent_id int4 NULL,
  tin_id varchar(32) NULL,
  new_register varchar(32) NULL,
  entity_type int4 NULL,
  civil_id int8 NULL,
  CONSTRAINT bs_person_tin_uk UNIQUE (tin_id),
  CONSTRAINT ps_person_pkey PRIMARY KEY (person_id),
  CONSTRAINT bs_person_entity_type_fkey FOREIGN KEY (entity_type) REFERENCES codelists.cl_legal_entity_type(id) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT bs_person_parent_fkey FOREIGN KEY (parent_id) REFERENCES base.bs_person(person_id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT bs_person_type_fkey FOREIGN KEY ("type") REFERENCES codelists.cl_person_type(code) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS bs_person_id_idx ON base.bs_person USING btree (person_id);
CREATE INDEX IF NOT EXISTS bs_person_person_register_idx ON base.bs_person USING btree (person_register);
CREATE INDEX IF NOT EXISTS idx_bs_person_register ON base.bs_person USING btree (upper((person_register)::text));
CREATE INDEX IF NOT EXISTS idx_bs_person_register_id ON base.bs_person USING btree (upper((person_register)::text), person_id);

DROP TRIGGER IF EXISTS audit_trigger_row ON base.bs_person;
CREATE TRIGGER audit_trigger_row AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
FOR EACH ROW EXECUTE FUNCTION audit.if_modified_func('false');
DROP TRIGGER IF EXISTS audit_trigger_stm ON base.bs_person;
CREATE TRIGGER audit_trigger_stm AFTER TRUNCATE ON base.bs_person
FOR EACH STATEMENT EXECUTE FUNCTION audit.if_modified_func('false');
DROP TRIGGER IF EXISTS notify_cud ON base.bs_person;
CREATE TRIGGER notify_cud AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
FOR EACH ROW EXECUTE FUNCTION base.notify_event_all();
DROP TRIGGER IF EXISTS person_log ON base.bs_person;
CREATE TRIGGER person_log AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
FOR EACH ROW EXECUTE FUNCTION base.log_changes();

CREATE TABLE IF NOT EXISTS sdplatform.sd_organization (
  id serial4 NOT NULL,
  "name" varchar(250) NULL,
  CONSTRAINT sd_organization_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_user (
  user_id serial4 NOT NULL,
  username varchar(50) NULL,
  email varchar(250) NULL,
  "password" varchar(150) NULL,
  firstname varchar(50) NULL,
  lastname varchar(50) NULL,
  photo varchar(250) NULL,
  remember_token varchar(100) NULL,
  created_at date NULL,
  updated_at timestamp NULL,
  deleted_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  last_active timestamptz NULL,
  gis_user_real varchar(20) NULL DEFAULT NULL::character varying,
  is_active bool NULL DEFAULT false,
  CONSTRAINT sd_user_pkey PRIMARY KEY (user_id)
);

CREATE SEQUENCE IF NOT EXISTS sdplatform.sd_role_id_seq;

CREATE TABLE IF NOT EXISTS sdplatform.sd_role (
  role_id int4 NOT NULL DEFAULT nextval('sdplatform.sd_role_id_seq'::regclass),
  code varchar(10) NULL DEFAULT NULL::character varying,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  description text NULL,
  role_type_code varchar(45) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  editor_user int4 NULL,
  CONSTRAINT sd_role_pkey PRIMARY KEY (role_id),
  CONSTRAINT sd_role_uk UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_user_role (
  user_id int4 NOT NULL,
  role_id int4 NOT NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  CONSTRAINT sd_user_role_pkey PRIMARY KEY (user_id, role_id)
);

DROP TRIGGER IF EXISTS user_role_log ON sdplatform.sd_user_role;
CREATE TRIGGER user_role_log AFTER INSERT OR DELETE OR UPDATE ON sdplatform.sd_user_role
FOR EACH ROW EXECUTE FUNCTION base.log_changes();

CREATE TABLE IF NOT EXISTS sdplatform.sd_system (
  system_id serial4 NOT NULL,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  description text NULL,
  CONSTRAINT pk_sd_system PRIMARY KEY (system_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_permission (
  permission_id serial4 NOT NULL,
  code varchar(45) NULL DEFAULT NULL::character varying,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  CONSTRAINT sd_permission_pkey PRIMARY KEY (permission_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_menu (
  menu_id serial4 NOT NULL,
  code varchar(32) NULL DEFAULT NULL::character varying,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  parent_id int4 NULL,
  menu_url varchar(250) NULL DEFAULT NULL::character varying,
  sort_order int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  menu_icon varchar(150) NULL DEFAULT NULL::character varying,
  parent_code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_menu_pkey PRIMARY KEY (menu_id),
  CONSTRAINT sd_menuu_uk UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_system_role (
  system_role_id serial4 NOT NULL,
  system_id int4 NULL,
  role_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  CONSTRAINT pk_sd_system_role PRIMARY KEY (system_role_id),
  CONSTRAINT fk_sd_system_role_sd_role FOREIGN KEY (role_id) REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_sd_system_role_sd_system FOREIGN KEY (system_id) REFERENCES sdplatform.sd_system(system_id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_role_permission (
  role_permission_id serial4 NOT NULL,
  role_id int4 NULL,
  permission_id int4 NULL,
  menu_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  CONSTRAINT sd_role_permission_pkey PRIMARY KEY (role_permission_id),
  CONSTRAINT sd_role_permission_fkey1 FOREIGN KEY (menu_id) REFERENCES sdplatform.sd_menu(menu_id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT sd_role_permission_fkey2 FOREIGN KEY (permission_id) REFERENCES sdplatform.sd_permission(permission_id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT sd_role_permission_fkey3 FOREIGN KEY (role_id) REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_menu_role (
  menu_id int4 NOT NULL,
  role_id int4 NOT NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  CONSTRAINT sd_menu_role_pkey PRIMARY KEY (menu_id, role_id),
  CONSTRAINT sd_menu_role_fkey1 FOREIGN KEY (menu_id) REFERENCES sdplatform.sd_menu(menu_id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT sd_menu_role_fkey2 FOREIGN KEY (role_id) REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_form (
  form_id serial4 NOT NULL,
  form_name varchar(250) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_form_pkey PRIMARY KEY (form_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_menu_form (
  menu_form_id serial4 NOT NULL,
  form_id int4 NULL,
  menu_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  form_code varchar(20) NULL DEFAULT NULL::character varying,
  menu_code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_menu_form_pkey PRIMARY KEY (menu_form_id),
  CONSTRAINT sd_menu_form_fkey1 FOREIGN KEY (menu_id) REFERENCES sdplatform.sd_menu(menu_id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT sd_menu_form_fkey2 FOREIGN KEY (form_id) REFERENCES sdplatform.sd_form(form_id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_position (
  position_id serial4 NOT NULL,
  code varchar(45) NULL DEFAULT NULL::character varying,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  organization int4 NULL,
  department int4 NULL,
  is_meeting bool NOT NULL DEFAULT false,
  CONSTRAINT hr_employee_uk UNIQUE (code),
  CONSTRAINT hr_position_pkey PRIMARY KEY (position_id)
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_department (
  department_id serial4 NOT NULL,
  code varchar(45) NULL DEFAULT NULL::character varying,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  parent_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  organization int4 NULL,
  address varchar(250) NULL,
  fax varchar(25) NULL,
  phone varchar(50) NULL,
  website varchar(250) NULL,
  report_email varchar(250) NULL,
  au2 varchar(5) NULL DEFAULT NULL::character varying,
  au1 varchar(3) NULL DEFAULT NULL::character varying,
  bank_name varchar(250) NULL,
  head_surname varchar(75) NULL,
  head_firstname varchar(75) NULL,
  account_no varchar(50) NULL,
  bank_id int4 NULL,
  entity_id int4 NULL,
  right_types _varchar NULL DEFAULT '{}'::character varying[],
  au_work_zone int4 NULL,
  free_zone int4 NULL,
  only_lm bool NOT NULL DEFAULT false,
  CONSTRAINT hr_department_pkey PRIMARY KEY (department_id),
  CONSTRAINT hr_department_fkey1 FOREIGN KEY (parent_id) REFERENCES sdplatform.hr_department(department_id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT hr_department_fkey2 FOREIGN KEY (entity_id) REFERENCES base.bs_person(person_id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT sd_organization_fk FOREIGN KEY (organization) REFERENCES sdplatform.sd_organization(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sdplatform.hr_employee (
  employee_id serial4 NOT NULL,
  firstname varchar(250) NULL DEFAULT NULL::character varying,
  lastname varchar(250) NULL DEFAULT NULL::character varying,
  urag_name varchar(250) NULL DEFAULT NULL::character varying,
  mobile_phone varchar(50) NULL DEFAULT NULL::character varying,
  department_id int4 NULL,
  position_id int4 NULL,
  employee_type_id int4 NULL,
  user_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  hired_date date NULL,
  terminated_date date NULL,
  is_active int4 NULL,
  email varchar(250) NULL DEFAULT NULL::character varying,
  home_phone varchar(45) NULL DEFAULT NULL::character varying,
  profile_photo text NULL DEFAULT NULL::character varying,
  time_attendance_code varchar(45) NULL DEFAULT NULL::character varying,
  person_id int4 NULL,
  register bpchar(1) NULL,
  authentication_decision_no bpchar(256) NULL,
  expired_decision_no bpchar(256) NULL,
  hired_decision_no varchar(45) NULL,
  terminated_decision_no varchar(45) NULL,
  authentication_date date NULL,
  expired_date date NULL,
  is_active_request bool NULL DEFAULT false,
  CONSTRAINT hr_employee_pkey PRIMARY KEY (employee_id),
  CONSTRAINT hr_employee_fkey1 FOREIGN KEY (person_id) REFERENCES base.bs_person(person_id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT hr_employee_fkey2 FOREIGN KEY (user_id) REFERENCES sdplatform.sd_user(user_id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT hr_employee_person_id_fkey FOREIGN KEY (person_id) REFERENCES base.bs_person(person_id) ON DELETE RESTRICT ON UPDATE CASCADE
);

INSERT INTO sdplatform.sd_system (system_id, "name", description)
VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
ON CONFLICT (system_id) DO UPDATE SET
  "name" = EXCLUDED."name",
  description = EXCLUDED.description;

INSERT INTO sdplatform.sd_permission (code, "name", created_at)
SELECT v.code, v.name, NOW()
FROM (VALUES
  ('A001', 'land:read', NOW()),
  ('A002', 'land:create', NOW()),
  ('A003', 'land:update', NOW()),
  ('A004', 'land:delete', NOW()),
  ('A005', 'compensation:read', NOW()),
  ('A006', 'compensation:create', NOW()),
  ('A007', 'compensation:update', NOW()),
  ('A008', 'decision:read', NOW()),
  ('A009', 'decision:create', NOW()),
  ('A010', 'decision:update', NOW()),
  ('A011', 'decision:delete', NOW())
) AS v(code, name, created_at)
WHERE NOT EXISTS (
  SELECT 1
  FROM sdplatform.sd_permission p
  WHERE p.code = v.code OR p."name" = v.name
);

INSERT INTO sdplatform.sd_role (code, "name", description, created_at)
VALUES
  ('A100', 'admin', 'Газар чөлөөлөлтийн системийн хэрэглэгч үүсгэх админ', NOW()),
  ('A101', 'senior_specialist', 'Ахлах мэргэжилтэн', NOW()),
  ('A102', 'employee', 'Энгийн ажилтан', NOW())
ON CONFLICT (code) DO UPDATE SET
  "name" = EXCLUDED."name",
  description = EXCLUDED.description,
  updated_at = NOW();

INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
SELECT 10, r.role_id, NOW()
FROM sdplatform.sd_role r
WHERE r.code IN ('A100', 'A101', 'A102')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_system_role sr
    WHERE sr.system_id = 10 AND sr.role_id = r.role_id
  );

INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at, parent_code)
VALUES
  ('A010', 'Газар чөлөөлөлт', '/land-acquisitions', 1, NOW(), NULL),
  ('A010001', 'Газар чөлөөлөлтийн жагсаалт', '/land-acquisitions', 1, NOW(), 'A010'),
  ('A010002', 'Нэгж талбар', '/land-acquisitions/:id/parcels', 2, NOW(), 'A010'),
  ('A010003', 'Хөрөнгө, үнэлгээ', '/land-acquisitions/:id/assets', 3, NOW(), 'A010'),
  ('A010004', 'Нөхөх олговор', '/land-acquisitions/:id/compensations', 4, NOW(), 'A010'),
  ('A010005', 'Баримт бичиг', '/land-acquisitions/:id/documents', 5, NOW(), 'A010')
ON CONFLICT (code) DO UPDATE SET
  "name" = EXCLUDED."name",
  menu_url = EXCLUDED.menu_url,
  sort_order = EXCLUDED.sort_order,
  parent_code = EXCLUDED.parent_code,
  updated_at = NOW();

UPDATE sdplatform.sd_menu child
SET parent_id = parent.menu_id
FROM sdplatform.sd_menu parent
WHERE child.parent_code = parent.code;

INSERT INTO sdplatform.sd_form (form_name, code, created_at)
SELECT m."name", m.code, NOW()
FROM sdplatform.sd_menu m
WHERE m.code IN ('A010', 'A010001', 'A010002', 'A010003', 'A010004', 'A010005')
  AND NOT EXISTS (SELECT 1 FROM sdplatform.sd_form f WHERE f.code = m.code);

INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at)
SELECT m.menu_id, r.role_id, NOW()
FROM sdplatform.sd_menu m
CROSS JOIN sdplatform.sd_role r
WHERE m.code IN ('A010', 'A010001', 'A010002', 'A010003', 'A010004', 'A010005')
  AND r.code IN ('A100', 'A101', 'A102')
ON CONFLICT DO NOTHING;

INSERT INTO sdplatform.sd_menu_form (form_id, menu_id, form_code, menu_code, created_at)
SELECT f.form_id, m.menu_id, f.code, m.code, NOW()
FROM sdplatform.sd_form f
JOIN sdplatform.sd_menu m ON m.code = f.code
WHERE m.code IN ('A010', 'A010001', 'A010002', 'A010003', 'A010004', 'A010005')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_menu_form mf
    WHERE mf.form_code = f.code AND mf.menu_code = m.code
  );

INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at)
SELECT r.role_id, p.permission_id, m.menu_id, NOW()
FROM sdplatform.sd_role r
JOIN sdplatform.sd_permission p ON p."name" IN (
  'land:read', 'land:create', 'land:update', 'land:delete',
  'compensation:read', 'compensation:create', 'compensation:update',
  'decision:read', 'decision:create', 'decision:update', 'decision:delete'
)
JOIN sdplatform.sd_menu m ON m.code IN ('A010', 'A010001', 'A010002', 'A010003', 'A010004', 'A010005')
WHERE r.code IN ('A100', 'A101', 'A102')
  AND NOT EXISTS (
    SELECT 1 FROM sdplatform.sd_role_permission rp
    WHERE rp.role_id = r.role_id
      AND rp.permission_id = p.permission_id
      AND rp.menu_id = m.menu_id
  );

INSERT INTO sdplatform.sd_user (
  username,
  email,
  "password",
  firstname,
  lastname,
  created_at,
  updated_at,
  deleted_at,
  is_active
)
SELECT
  LEFT(u.username, 50),
  LEFT(u.email, 250),
  LEFT(u.password, 150),
  LEFT(u.first_name, 50),
  LEFT(u.last_name, 50),
  u.created_at::date,
  u.updated_at::timestamp,
  u.deleted_at::timestamp,
  u.is_active
FROM users u
WHERE NOT EXISTS (
  SELECT 1
  FROM sdplatform.sd_user su
  WHERE su.username = LEFT(u.username, 50)
     OR su.email = LEFT(u.email, 250)
);

INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
SELECT su.user_id, sr.role_id, NOW(), NOW()
FROM users u
JOIN sdplatform.sd_user su ON su.email = LEFT(u.email, 250)
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
JOIN sdplatform.sd_role sr ON sr."name" = r.name
ON CONFLICT DO NOTHING;
