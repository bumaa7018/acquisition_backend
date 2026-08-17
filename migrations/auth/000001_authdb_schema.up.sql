-- authdb-ийн БҮХ хүснэгт — нэг файлд.
--
-- ДҮРЭМ: миграц нь ЗӨВХӨН бүтэц үүсгэнэ. Хүснэгт/схем/индекс/trigger БАЙХГҮЙ
-- бол үүсгэнэ, байгаа бол юу ч хийхгүй өнгөрнө (CREATE ... IF NOT EXISTS).
-- Багана нэмэх/өөрчлөх, өгөгдөл нэмэх/засах/устгах ЭНД БАЙХГҮЙ:
--   * өгөгдөл  → seeder (зөвхөн байхгүй бол нэмнэ)
--   * өөрчлөлт → систем ассаны дараа API-аар л
--
-- Ингэснээр холболтыг жинхэнэ sdplatform өгөгдлийн сан руу залгахад тэр санд
-- байгаа хүснэгт, багана, өгөгдөл ЯМАР Ч БАЙДЛААР хөндөгдөхгүй.
--
--
-- ЖИЧ: trigger болон trigger-функцийг ДАРЖ БИЧИХГҮЙ. Жинхэнэ sdplatform санд
-- base.log_changes / base.notify_event_all / audit.if_modified_func нь БОДИТ
-- биетэй (аудит бүртгэл, notify) байдаг тул CREATE OR REPLACE хийвэл тэднийг
-- хоосон stub-аар дарж, тэр системийн бүртгэлийг чимээгүй унтраана. Иймд
-- зөвхөн БАЙХГҮЙ үед л үүсгэнэ (DROP TRIGGER огт хэрэглэхгүй).
--
-- Хүснэгтүүд: codelists/base/sdplatform схемүүдэд шаардлагатай хүснэгтүүд.
-- FK хамаарлын дарааллаар бичигдсэн.

-- ── extensions_schemas_functions ──────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE SCHEMA IF NOT EXISTS sdplatform;
CREATE SCHEMA IF NOT EXISTS base;
CREATE SCHEMA IF NOT EXISTS codelists;
CREATE SCHEMA IF NOT EXISTS audit;

DO $ensure_fn$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'base' AND p.proname = 'log_changes'
  ) THEN
    CREATE FUNCTION base.log_changes() RETURNS trigger LANGUAGE plpgsql AS $body$
    BEGIN
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
    END $body$;
  END IF;
END $ensure_fn$;

DO $ensure_fn$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'base' AND p.proname = 'notify_event_all'
  ) THEN
    CREATE FUNCTION base.notify_event_all() RETURNS trigger LANGUAGE plpgsql AS $body$
    BEGIN
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
    END $body$;
  END IF;
END $ensure_fn$;

DO $ensure_fn$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'audit' AND p.proname = 'if_modified_func'
  ) THEN
    CREATE FUNCTION audit.if_modified_func() RETURNS trigger LANGUAGE plpgsql AS $body$
    BEGIN
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
    END $body$;
  END IF;
END $ensure_fn$;

-- ── codelists_cl_person_type ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS codelists.cl_person_type (
  code int4 NOT NULL,
  description varchar(75) NOT NULL,
  description_en varchar(75) NULL,
  CONSTRAINT cl_person_type_description_en_key UNIQUE (description_en),
  CONSTRAINT cl_person_type_description_key UNIQUE (description),
  CONSTRAINT cl_person_type_pkey PRIMARY KEY (code)
);

-- ── codelists_cl_legal_entity_type ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS codelists.cl_legal_entity_type (
  id int4 NOT NULL,
  description varchar(250) NULL,
  CONSTRAINT cl_legal_entity_type_pkey PRIMARY KEY (id)
);

-- ── base_bs_person ──────────────────────────────────────────────
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

DO $ensure_tg$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'audit_trigger_row' AND tgrelid = 'base.bs_person'::regclass AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER audit_trigger_row AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
    FOR EACH ROW EXECUTE FUNCTION audit.if_modified_func('false');
  END IF;
END $ensure_tg$;
DO $ensure_tg$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'audit_trigger_stm' AND tgrelid = 'base.bs_person'::regclass AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER audit_trigger_stm AFTER TRUNCATE ON base.bs_person
    FOR EACH STATEMENT EXECUTE FUNCTION audit.if_modified_func('false');
  END IF;
END $ensure_tg$;
DO $ensure_tg$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'notify_cud' AND tgrelid = 'base.bs_person'::regclass AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER notify_cud AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
    FOR EACH ROW EXECUTE FUNCTION base.notify_event_all();
  END IF;
END $ensure_tg$;
DO $ensure_tg$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'person_log' AND tgrelid = 'base.bs_person'::regclass AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER person_log AFTER INSERT OR DELETE OR UPDATE ON base.bs_person
    FOR EACH ROW EXECUTE FUNCTION base.log_changes();
  END IF;
END $ensure_tg$;

-- ── sdplatform_sd_organization ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sdplatform.sd_organization (
  id serial4 NOT NULL,
  "name" varchar(250) NULL,
  CONSTRAINT sd_organization_pkey PRIMARY KEY (id)
);

-- ── sdplatform_sd_user ──────────────────────────────────────────────
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

-- ── sdplatform_sd_role ──────────────────────────────────────────────
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

-- ── sdplatform_sd_user_role ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sdplatform.sd_user_role (
  user_id int4 NOT NULL,
  role_id int4 NOT NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  created_by int4 NULL,
  updated_by int4 NULL,
  CONSTRAINT sd_user_role_pkey PRIMARY KEY (user_id, role_id)
);

DO $ensure_tg$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'user_role_log' AND tgrelid = 'sdplatform.sd_user_role'::regclass AND NOT tgisinternal
  ) THEN
    CREATE TRIGGER user_role_log AFTER INSERT OR DELETE OR UPDATE ON sdplatform.sd_user_role
    FOR EACH ROW EXECUTE FUNCTION base.log_changes();
  END IF;
END $ensure_tg$;

-- ── sdplatform_sd_system ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sdplatform.sd_system (
  system_id serial4 NOT NULL,
  "name" varchar(250) NULL DEFAULT NULL::character varying,
  description text NULL,
  CONSTRAINT pk_sd_system PRIMARY KEY (system_id)
);

-- ── sdplatform_sd_permission ──────────────────────────────────────────────
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

-- ── sdplatform_sd_menu ──────────────────────────────────────────────
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

-- ── sdplatform_sd_system_role ──────────────────────────────────────────────
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

-- ── sdplatform_sd_role_permission ──────────────────────────────────────────────
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

-- ── sdplatform_sd_menu_role ──────────────────────────────────────────────
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

-- ── sdplatform_sd_form ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sdplatform.sd_form (
  form_id serial4 NOT NULL,
  form_name varchar(250) NULL DEFAULT NULL::character varying,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_form_pkey PRIMARY KEY (form_id)
);

-- ── sdplatform_sd_menu_form ──────────────────────────────────────────────
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

-- ── sdplatform_sd_form_permission ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sdplatform.sd_form_permission (
  form_permission_id serial4 NOT NULL,
  form_id int4 NULL,
  permission_id int4 NULL,
  created_at timestamp NULL,
  updated_at timestamp NULL,
  form_code varchar(20) NULL DEFAULT NULL::character varying,
  permission_code varchar(20) NULL DEFAULT NULL::character varying,
  CONSTRAINT sd_form_permission_pkey PRIMARY KEY (form_permission_id)
);

-- ── sdplatform_hr_position ──────────────────────────────────────────────
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

-- ── sdplatform_hr_department ──────────────────────────────────────────────
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

-- ── sdplatform_hr_employee ──────────────────────────────────────────────
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

-- public.plan ЭНД БАЙХГҮЙ.
--
-- Төлөвлөгөөний мэдээлэл нь ЗӨВХӨН appdb дээр (public.plan) хадгалагдана,
-- тэр нь ЗӨВХӨН дундын сервисээс (middleware /plan/project) татсан датаг
-- агуулна. Уншилт ч appdb-ээс явна.
--
-- Өмнө нь энд ижил нэртэй хүснэгтүүд байж, төсөөллийн өгөгдлөөр seed
-- хийгддэг байсан — тэр нь жинхэнэ ГУС сан руу залгахад төөрөгдөл үүсгэнэ.
